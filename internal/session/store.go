// This file implements the on-disk session-token cache half of the
// package. The authoritative package overview lives in doc.go; the
// blank line below keeps this from being a second package-doc comment.

package session

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/gofrs/flock"
)

// lockRetry is the polling interval used by flock.LockContext while
// waiting for a contended lock. Small enough to feel responsive when
// another process drops the lock, large enough not to spin on syscalls.
const lockRetry = 50 * time.Millisecond

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

// Store persists KasAuth credential tokens between CLI invocations so a
// successful interactive login (including 2FA) is not repeated on every
// command for as long as the server keeps the session alive. It is a
// single TOML file (sessions.toml) keyed by login, living next to the
// config file and written with mode 0600. Entries expire client-side
// via ExpiresAt; the server's session_lifetime is the source of truth,
// but mirroring it locally lets the CLI skip the KasAuth round trip
// while a token is still good and refresh it transparently once it is
// not.
//
// The zero value is not usable; obtain one via New.
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

// PathFor returns the sessions file path that pairs with configPath.
// Empty configPath resolves to DefaultPath. Otherwise the file lives
// next to the config file as sessions.toml so a custom --config flag
// keeps both halves of the on-disk state colocated.
func PathFor(configPath string) (string, error) {
	if configPath == "" {
		return DefaultPath()
	}
	return filepath.Join(filepath.Dir(configPath), "sessions.toml"), nil
}

// LockPath returns the advisory-lock-file path that pairs with the
// sessions file. Two CLI processes serialise their read-modify-write
// cycle on this lock so a Heartbeat from one cannot silently overwrite
// a Save from another.
func (s *Store) LockPath() string {
	return s.Path + ".lock"
}

// withLock runs fn while holding an exclusive advisory lock on
// LockPath. The lock file is created next to sessions.toml so its
// permissions inherit from the same parent directory (created 0700
// by write). The lock is released even if fn panics. ctx cancels both
// the wait for the lock (via flock.LockContext) and the body via the
// caller's pre-I/O ctx.Err() checks.
func (s *Store) withLock(ctx context.Context, fn func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return fmt.Errorf("session: create lock dir: %w", err)
	}
	lock := flock.New(s.LockPath())
	locked, err := lock.TryLockContext(ctx, lockRetry)
	if err != nil {
		return fmt.Errorf("session: acquire lock %s: %w", s.LockPath(), err)
	}
	if !locked {
		return fmt.Errorf("session: acquire lock %s: %w", s.LockPath(), ctx.Err())
	}
	defer func() { _ = lock.Unlock() }()
	return fn()
}

// Load returns the entry stored for login. nil is returned (without
// error) when the file is absent, the login is not present, or the
// entry has expired; expired entries are best-effort removed.
//
// Load serialises with concurrent Save / Delete calls (including from
// other kasapi-cli processes) via an advisory file lock at LockPath so
// a concurrent write cannot tear the read-modify-write of an expiry
// cleanup. ctx cancels the wait for the lock and short-circuits the
// pre-I/O entry; the underlying toml/os calls are synchronous and not
// further interruptible, but the file is local so blocking is bounded.
func (s *Store) Load(ctx context.Context, login string) (*Entry, error) {
	if login == "" {
		return nil, errors.New("session: Load requires login")
	}
	var out *Entry
	err := s.withLock(ctx, func() error {
		file, err := s.read()
		if err != nil {
			return err
		}
		e, ok := file.Sessions[login]
		if !ok {
			return nil
		}
		// A zero expires_at can only come from a hand-edited file — Save
		// and Refresh always fill it. Treating it as never-expiring would
		// make the token locally immortal, so it is expired instead.
		if e.ExpiresAt.IsZero() || !s.now().Before(e.ExpiresAt) {
			return s.deleteLocked(login)
		}
		out = &e
		return nil
	})
	return out, err
}

// Save writes e under login, replacing any existing entry. If
// ExpiresAt is zero, it is computed as Now+LifetimeSeconds (or
// Now+DefaultLifetime when LifetimeSeconds is 0).
//
// Save serialises with concurrent Load / Delete / Save / Refresh calls
// (including from other kasapi-cli processes) via an advisory file lock
// at LockPath. Heartbeats must go through Refresh, not Save — Save
// replaces unconditionally and would clobber a newer token another
// process has persisted in the meantime. ctx cancels the wait for the
// lock.
func (s *Store) Save(ctx context.Context, login string, e Entry) error {
	if login == "" {
		return errors.New("session: Save requires login")
	}
	if e.Token == "" {
		return errors.New("session: Save requires token")
	}
	if e.ExpiresAt.IsZero() {
		e.ExpiresAt = s.now().Add(s.lifetime(e))
	}
	return s.withLock(ctx, func() error {
		file, err := s.read()
		if err != nil {
			return err
		}
		if file.Sessions == nil {
			file.Sessions = map[string]Entry{}
		}
		file.Sessions[login] = e
		return s.write(file)
	})
}

// Refresh persists e under login only while the stored entry still
// carries the same token as e.Token. It is the Heartbeat counterpart of
// Save: a heartbeat re-persists the token its process authenticated
// with plus a fresh expiry, so when another process has since saved a
// different (newer) token, writing the stale one back would clobber
// that update — the newer entry is left untouched instead. A missing
// file or entry is likewise left alone: there is nothing the stale
// token may extend. If ExpiresAt is zero it is computed as in Save.
//
// Refresh serialises with concurrent Load / Save / Delete calls via the
// advisory file lock at LockPath. ctx cancels the wait for the lock.
func (s *Store) Refresh(ctx context.Context, login string, e Entry) error {
	if login == "" {
		return errors.New("session: Refresh requires login")
	}
	if e.Token == "" {
		return errors.New("session: Refresh requires token")
	}
	if e.ExpiresAt.IsZero() {
		e.ExpiresAt = s.now().Add(s.lifetime(e))
	}
	return s.withLock(ctx, func() error {
		file, err := s.read()
		if err != nil {
			return err
		}
		cur, ok := file.Sessions[login]
		if !ok || cur.Token != e.Token {
			return nil
		}
		file.Sessions[login] = e
		return s.write(file)
	})
}

// Delete removes the entry for login. Missing files and missing
// entries are not errors. The file itself is removed when the last
// entry is taken out so the on-disk state matches "no sessions".
//
// Delete serialises with concurrent Load / Save / Delete calls via the
// advisory file lock at LockPath. ctx cancels the wait for the lock.
func (s *Store) Delete(ctx context.Context, login string) error {
	if login == "" {
		return errors.New("session: Delete requires login")
	}
	return s.withLock(ctx, func() error { return s.deleteLocked(login) })
}

// deleteLocked is the un-locked Delete body, callable from inside a
// withLock-protected block (e.g. by Load when an expired entry is
// dropped) without re-entering the lock.
func (s *Store) deleteLocked(login string) error {
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
	// Chmod is redundant on Unix where os.CreateTemp already opens the
	// file with mode 0600, but Windows ignores the create-mode bits, so
	// we set them explicitly as defense-in-depth across platforms.
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
