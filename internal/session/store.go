// Package session persists KasAuth credential tokens between CLI
// invocations so a successful interactive login (including 2FA) does
// not have to be repeated on every command for as long as the server
// keeps the session alive.
//
// The store is a single TOML file (sessions.toml) keyed by login,
// living next to the config file and written with mode 0600. Entries
// expire client-side via ExpiresAt; the server's session_lifetime is
// the source of truth, but mirroring it locally lets the CLI skip the
// KasAuth round trip while a token is still good and refresh it
// transparently once it is not.
package session

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

// DefaultLifetime is applied when an Entry is saved without a
// LifetimeSeconds value. It matches the documented KasAuth default
// (session_lifetime, 24 h) so a CLI invocation that did not pass
// --session-lifetime still ends up with a sensible local expiry.
const DefaultLifetime = 24 * time.Hour

// Entry is one cached session token. The map key in the on-disk file
// is the login, so Login is not encoded.
type Entry struct {
	Token           string    `toml:"token"`
	ExpiresAt       time.Time `toml:"expires_at"`
	LifetimeSeconds int       `toml:"lifetime_seconds"`
	UpdateLifetime  bool      `toml:"update_lifetime"`
}

// Store reads and writes sessions to a TOML file. The zero value is
// not usable; obtain one via New.
type Store struct {
	Path string
	Now  func() time.Time
}

// fileFormat mirrors the on-disk schema:
//
//	[sessions.<login>]
//	token = "..."
//	expires_at = 2026-05-03T12:00:00Z
//	lifetime_seconds = 3600
//	update_lifetime = true
type fileFormat struct {
	Sessions map[string]Entry `toml:"sessions"`
}

// New returns a Store backed by path. Empty path resolves to
// DefaultPath. The clock is wall-clock; tests override Now.
func New(path string) (*Store, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return nil, err
		}
	}
	return &Store{Path: path, Now: time.Now}, nil
}

// DefaultPath returns the OS-specific default location of the
// sessions file, alongside the config file.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("session: locate user config dir: %w", err)
	}
	return filepath.Join(dir, "kasapi-cli", "sessions.toml"), nil
}

// Load returns the entry stored for login. nil is returned (without
// error) when the file is absent, the login is not present, or the
// entry has expired; expired entries are best-effort removed.
func (s *Store) Load(login string) (*Entry, error) {
	if login == "" {
		return nil, errors.New("session: Load requires login")
	}
	file, err := s.read()
	if err != nil {
		return nil, err
	}
	e, ok := file.Sessions[login]
	if !ok {
		return nil, nil
	}
	if !e.ExpiresAt.IsZero() && !s.now().Before(e.ExpiresAt) {
		_ = s.Delete(login)
		return nil, nil
	}
	return &e, nil
}

// Save writes e under login, replacing any existing entry. If
// ExpiresAt is zero, it is computed as Now+LifetimeSeconds (or
// Now+DefaultLifetime when LifetimeSeconds is 0).
func (s *Store) Save(login string, e Entry) error {
	if login == "" {
		return errors.New("session: Save requires login")
	}
	if e.Token == "" {
		return errors.New("session: Save requires token")
	}
	if e.ExpiresAt.IsZero() {
		e.ExpiresAt = s.now().Add(s.lifetime(e))
	}
	file, err := s.read()
	if err != nil {
		return err
	}
	if file.Sessions == nil {
		file.Sessions = map[string]Entry{}
	}
	file.Sessions[login] = e
	return s.write(file)
}

// Delete removes the entry for login. Missing files and missing
// entries are not errors. The file itself is removed when the last
// entry is taken out so the on-disk state matches "no sessions".
func (s *Store) Delete(login string) error {
	if login == "" {
		return errors.New("session: Delete requires login")
	}
	file, err := s.read()
	if err != nil {
		return err
	}
	if _, ok := file.Sessions[login]; !ok {
		return nil
	}
	delete(file.Sessions, login)
	if len(file.Sessions) == 0 {
		if rerr := os.Remove(s.Path); rerr != nil && !errors.Is(rerr, os.ErrNotExist) {
			return fmt.Errorf("session: remove %s: %w", s.Path, rerr)
		}
		return nil
	}
	return s.write(file)
}

func (s *Store) read() (fileFormat, error) {
	var file fileFormat
	if _, err := toml.DecodeFile(s.Path, &file); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fileFormat{}, nil
		}
		return fileFormat{}, fmt.Errorf("session: parse %s: %w", s.Path, err)
	}
	return file, nil
}

func (s *Store) write(file fileFormat) error {
	dir := filepath.Dir(s.Path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("session: create dir %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".sessions-*.toml")
	if err != nil {
		return fmt.Errorf("session: create temp: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }
	if cherr := tmp.Chmod(0o600); cherr != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("session: chmod %s: %w", tmpPath, cherr)
	}
	if encErr := encode(tmp, file); encErr != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("session: encode: %w", encErr)
	}
	if cerr := tmp.Close(); cerr != nil {
		cleanup()
		return fmt.Errorf("session: close temp: %w", cerr)
	}
	if rerr := os.Rename(tmpPath, s.Path); rerr != nil {
		cleanup()
		return fmt.Errorf("session: rename to %s: %w", s.Path, rerr)
	}
	return nil
}

func encode(w io.Writer, file fileFormat) error {
	return toml.NewEncoder(w).Encode(file)
}

func (s *Store) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Store) lifetime(e Entry) time.Duration {
	if e.LifetimeSeconds > 0 {
		return time.Duration(e.LifetimeSeconds) * time.Second
	}
	return DefaultLifetime
}
