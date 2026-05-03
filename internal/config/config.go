package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Auth type values accepted by the KAS API.
const (
	AuthPlain   = "plain"
	AuthSession = "session"
)

// Profile holds the credentials for a single KAS account.
type Profile struct {
	Login    string `toml:"login"`
	AuthData string `toml:"auth_data"`
	AuthType string `toml:"auth_type"`
}

// Config is the parsed TOML file.
type Config struct {
	DefaultProfile string             `toml:"default_profile"`
	Profiles       map[string]Profile `toml:"profiles"`
}

// ErrNoConfig is returned by Load when the config file does not exist.
// Callers may continue with a nil Config and rely on flags or env vars.
var ErrNoConfig = errors.New("config: file not found")

// DefaultPath returns the OS-specific default location of the config
// file: $XDG_CONFIG_HOME/kasapi-cli/config.toml on Linux, equivalent
// paths on macOS and Windows via os.UserConfigDir.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("config: locate user config dir: %w", err)
	}
	return filepath.Join(dir, "kasapi-cli", "config.toml"), nil
}

// Load reads and parses the config file at path. If path is empty the
// OS default location (DefaultPath) is used. ErrNoConfig is returned
// when the file does not exist; other errors signal malformed input.
func Load(path string) (*Config, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return nil, err
		}
	}
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNoConfig
		}
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config: %s: %w", path, err)
	}
	return &cfg, nil
}

// Save writes c as TOML to path. Empty path resolves to DefaultPath.
// Parent directories are created with mode 0700; the file is written
// with mode 0600. The write goes through a temp file in the same
// directory and an atomic rename so a crash mid-write cannot corrupt
// an existing config.
func (c *Config) Save(path string) error {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return err
		}
	}
	if err := c.validate(); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("config: create dir %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".config-*.toml")
	if err != nil {
		return fmt.Errorf("config: create temp: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("config: chmod %s: %w", tmpPath, err)
	}
	if err := c.encode(tmp); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("config: encode: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("config: close temp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		cleanup()
		return fmt.Errorf("config: rename to %s: %w", path, err)
	}
	return nil
}

func (c *Config) encode(w io.Writer) error {
	return toml.NewEncoder(w).Encode(c)
}

func (c *Config) validate() error {
	for name, p := range c.Profiles {
		if p.AuthType != "" && p.AuthType != AuthPlain && p.AuthType != AuthSession {
			return fmt.Errorf("profile %q: auth_type %q must be %q or %q", name, p.AuthType, AuthPlain, AuthSession)
		}
	}
	if c.DefaultProfile != "" {
		if _, ok := c.Profiles[c.DefaultProfile]; !ok {
			return fmt.Errorf("default_profile %q not defined under [profiles]", c.DefaultProfile)
		}
	}
	return nil
}
