package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/config"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestLoadValid(t *testing.T) {
	path := writeConfig(t, `
default_profile = "main"

[profiles.main]
login     = "w0000000"
auth_data = "secret"
auth_type = "session"

[profiles.staging]
login     = "w0000001"
auth_data = "other"
auth_type = "plain"
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DefaultProfile != "main" {
		t.Errorf("DefaultProfile = %q", cfg.DefaultProfile)
	}
	if len(cfg.Profiles) != 2 {
		t.Errorf("len(Profiles) = %d, want 2", len(cfg.Profiles))
	}
	if cfg.Profiles["staging"].AuthType != "plain" {
		t.Errorf("staging.AuthType = %q", cfg.Profiles["staging"].AuthType)
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := config.Load(filepath.Join(t.TempDir(), "absent.toml"))
	if !errors.Is(err, config.ErrNoConfig) {
		t.Fatalf("err = %v, want ErrNoConfig", err)
	}
}

func TestLoadMalformed(t *testing.T) {
	path := writeConfig(t, "default_profile = \nthis is = not toml [[")
	_, err := config.Load(path)
	if err == nil || errors.Is(err, config.ErrNoConfig) {
		t.Fatalf("expected parse error, got %v", err)
	}
}

func TestLoadRejectsUnknownAuthType(t *testing.T) {
	path := writeConfig(t, `
[profiles.main]
login     = "x"
auth_data = "y"
auth_type = "magic"
`)
	_, err := config.Load(path)
	if err == nil || !strings.Contains(err.Error(), "auth_type") {
		t.Fatalf("expected auth_type validation error, got %v", err)
	}
}

func TestLoadRejectsUnknownDefaultProfile(t *testing.T) {
	path := writeConfig(t, `
default_profile = "nope"

[profiles.main]
login     = "x"
auth_data = "y"
auth_type = "session"
`)
	_, err := config.Load(path)
	if err == nil || !strings.Contains(err.Error(), "default_profile") {
		t.Fatalf("expected default_profile validation error, got %v", err)
	}
}

func sampleConfig() *config.Config {
	return &config.Config{
		DefaultProfile: "main",
		Profiles: map[string]config.Profile{
			"main": {
				Login:    "w0000000",
				AuthData: "main-secret",
				AuthType: config.AuthSession,
			},
			"staging": {
				Login:    "w0000001",
				AuthData: "stg-secret",
				AuthType: config.AuthPlain,
			},
		},
	}
}

func TestResolvePrecedence(t *testing.T) {
	cfg := sampleConfig()
	cases := []struct {
		name string
		env  config.Env
		ov   config.Override
		want config.Credentials
	}{
		{
			name: "default profile",
			want: config.Credentials{Login: "w0000000", AuthData: "main-secret", AuthType: "session"},
		},
		{
			name: "named profile via flag",
			ov:   config.Override{Profile: "staging"},
			want: config.Credentials{Login: "w0000001", AuthData: "stg-secret", AuthType: "plain"},
		},
		{
			name: "env overrides profile",
			env:  config.Env{Login: "envuser", AuthData: "env-secret"},
			want: config.Credentials{Login: "envuser", AuthData: "env-secret", AuthType: "session"},
		},
		{
			name: "flag overrides env and profile",
			env:  config.Env{Login: "envuser"},
			ov:   config.Override{Login: "flaguser", AuthData: "flag-secret", AuthType: "plain"},
			want: config.Credentials{Login: "flaguser", AuthData: "flag-secret", AuthType: "plain"},
		},
		{
			name: "partial flag falls back per-field",
			ov:   config.Override{Login: "flaguser"},
			want: config.Credentials{Login: "flaguser", AuthData: "main-secret", AuthType: "session"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := cfg.Resolve(tc.env, tc.ov)
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestResolveUnknownProfile(t *testing.T) {
	cfg := sampleConfig()
	_, err := cfg.Resolve(config.Env{}, config.Override{Profile: "missing"})
	if err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("expected unknown-profile error, got %v", err)
	}
}

func TestResolveMissingCredentials(t *testing.T) {
	cfg := &config.Config{
		Profiles: map[string]config.Profile{
			"only-login": {Login: "w0000000"},
		},
		DefaultProfile: "only-login",
	}
	_, err := cfg.Resolve(config.Env{}, config.Override{})
	if err == nil {
		t.Fatal("expected error for missing credentials")
	}
	for _, want := range []string{"auth_data", "auth_type"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

func TestResolveEnvOnlyNoConfig(t *testing.T) {
	var cfg *config.Config
	got, err := cfg.Resolve(config.Env{
		Login:    "w0000000",
		AuthData: "secret",
		AuthType: "session",
	}, config.Override{})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want := config.Credentials{Login: "w0000000", AuthData: "secret", AuthType: "session"}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestResolveProfileWithoutConfigRejected(t *testing.T) {
	var cfg *config.Config
	_, err := cfg.Resolve(config.Env{}, config.Override{Profile: "main"})
	if err == nil || !strings.Contains(err.Error(), "no config file") {
		t.Fatalf("expected error about missing config, got %v", err)
	}
}

func TestCredentialsStringRedacts(t *testing.T) {
	c := config.Credentials{Login: "w0000000", AuthData: "supersecret", AuthType: "session"}
	s := c.String()
	if strings.Contains(s, "supersecret") {
		t.Errorf("AuthData leaked: %q", s)
	}
	if !strings.Contains(s, "redacted") {
		t.Errorf("missing redaction marker: %q", s)
	}
	if !strings.Contains(s, "w0000000") {
		t.Errorf("Login should be visible: %q", s)
	}
}

func TestCredentialsStringEmptyAuth(t *testing.T) {
	c := config.Credentials{Login: "w0000000"}
	s := c.String()
	if !strings.Contains(s, "<unset>") {
		t.Errorf("empty AuthData should render as <unset>, got %q", s)
	}
}

func TestEnvFromOS(t *testing.T) {
	t.Setenv("KAS_LOGIN", "envuser")
	t.Setenv("KAS_AUTHDATA", "envsecret")
	t.Setenv("KAS_AUTHTYPE", "plain")
	got := config.EnvFromOS()
	want := config.Env{Login: "envuser", AuthData: "envsecret", AuthType: "plain"}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}
