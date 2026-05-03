package config

import (
	"errors"
	"fmt"
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
