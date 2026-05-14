package cli_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/cli"
)

func TestBuildAPIClientPlain(t *testing.T) {
	t.Setenv("KAS_LOGIN", "")
	t.Setenv("KAS_AUTHDATA", "")
	t.Setenv("KAS_AUTHTYPE", "")

	opts := &cli.RootOptions{
		ConfigPath: filepath.Join(t.TempDir(), "missing-config.toml"),
		Login:      "w0000000",
		AuthData:   "secret-password",
		AuthType:   "plain",
	}
	c, err := cli.BuildAPIClient(opts)
	if err != nil {
		t.Fatalf("BuildAPIClient: %v", err)
	}
	if c == nil || c.Tokens == nil || c.Transport == nil {
		t.Fatalf("client incomplete: %+v", c)
	}
}

func TestBuildAPIClientSession(t *testing.T) {
	t.Setenv("KAS_LOGIN", "")
	t.Setenv("KAS_AUTHDATA", "")
	t.Setenv("KAS_AUTHTYPE", "")

	opts := &cli.RootOptions{
		ConfigPath: filepath.Join(t.TempDir(), "missing-config.toml"),
		Login:      "w0000000",
		AuthData:   "secret-password",
		AuthType:   "session",
	}
	c, err := cli.BuildAPIClient(opts)
	if err != nil {
		t.Fatalf("BuildAPIClient: %v", err)
	}
	if c == nil || c.Tokens == nil {
		t.Fatalf("client incomplete: %+v", c)
	}
}

func TestBuildAPIClientMissingCreds(t *testing.T) {
	t.Setenv("KAS_LOGIN", "")
	t.Setenv("KAS_AUTHDATA", "")
	t.Setenv("KAS_AUTHTYPE", "")

	opts := &cli.RootOptions{
		ConfigPath: filepath.Join(t.TempDir(), "missing-config.toml"),
	}
	_, err := cli.BuildAPIClient(opts)
	if err == nil {
		t.Fatal("expected error for missing credentials")
	}
	var ee *cli.ExitError
	if !errors.As(err, &ee) {
		t.Errorf("err is not *ExitError: %T", err)
	} else if ee.Code != cli.ExitUserError {
		t.Errorf("Code = %d, want %d", ee.Code, cli.ExitUserError)
	}
	// Genuine first-run (no config file at all) must surface the
	// `config init` discoverability hint so a fresh user is pointed at
	// the bootstrap wizard instead of guessing flag combinations.
	if !strings.Contains(err.Error(), "config init") {
		t.Errorf("err = %q, want hint containing %q", err, "config init")
	}
}

func TestBuildAPIClientPartialConfigNoHint(t *testing.T) {
	t.Setenv("KAS_LOGIN", "")
	t.Setenv("KAS_AUTHDATA", "")
	t.Setenv("KAS_AUTHTYPE", "")

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	contents := "default_profile = \"broken\"\n\n[profiles.broken]\nlogin = \"w0000000\"\n"
	if err := os.WriteFile(cfgPath, []byte(contents), 0o600); err != nil {
		t.Fatalf("write partial config: %v", err)
	}

	opts := &cli.RootOptions{ConfigPath: cfgPath}
	_, err := cli.BuildAPIClient(opts)
	if err == nil {
		t.Fatal("expected error for partial config")
	}
	if strings.Contains(err.Error(), "config init") {
		t.Errorf("err = %q, must not contain hint %q for partial config", err, "config init")
	}
}

func TestBuildAPIClientNilOpts(t *testing.T) {
	_, err := cli.BuildAPIClient(nil)
	if err == nil {
		t.Fatal("expected error for nil opts")
	}
	var ee *cli.ExitError
	if !errors.As(err, &ee) || ee.Code != cli.ExitUserError {
		t.Errorf("got %v, want ExitUserError", err)
	}
}

func TestBuildAPIClientSessionWithOTP(t *testing.T) {
	t.Setenv("KAS_LOGIN", "")
	t.Setenv("KAS_AUTHDATA", "")
	t.Setenv("KAS_AUTHTYPE", "")

	opts := &cli.RootOptions{
		ConfigPath: filepath.Join(t.TempDir(), "missing-config.toml"),
		Login:      "w0000000",
		AuthData:   "secret-password",
		AuthType:   "session",
		OTP:        "123456",
	}
	c, err := cli.BuildAPIClient(opts)
	if err != nil {
		t.Fatalf("BuildAPIClient: %v", err)
	}
	if c == nil || c.Tokens == nil {
		t.Fatalf("client incomplete: %+v", c)
	}
}

func TestBuildAPIClientPlainWithOTPRejected(t *testing.T) {
	t.Setenv("KAS_LOGIN", "")
	t.Setenv("KAS_AUTHDATA", "")
	t.Setenv("KAS_AUTHTYPE", "")

	opts := &cli.RootOptions{
		ConfigPath: filepath.Join(t.TempDir(), "missing-config.toml"),
		Login:      "w0000000",
		AuthData:   "secret-password",
		AuthType:   "plain",
		OTP:        "123456",
	}
	_, err := cli.BuildAPIClient(opts)
	if err == nil {
		t.Fatal("expected error when --otp combined with auth_type=plain")
	}
	var ee *cli.ExitError
	if !errors.As(err, &ee) || ee.Code != cli.ExitUserError {
		t.Errorf("got %v, want ExitUserError", err)
	}
}

func TestBuildAPIClientSessionWithLifetimeFlags(t *testing.T) {
	t.Setenv("KAS_LOGIN", "")
	t.Setenv("KAS_AUTHDATA", "")
	t.Setenv("KAS_AUTHTYPE", "")

	for _, ulv := range []string{"Y", "N", ""} {
		ulv := ulv
		t.Run("update-lifetime="+ulv, func(t *testing.T) {
			t.Parallel()
			opts := &cli.RootOptions{
				ConfigPath:            filepath.Join(t.TempDir(), "missing-config.toml"),
				Login:                 "w0000000",
				AuthData:              "secret-password",
				AuthType:              "session",
				SessionLifetime:       1800,
				SessionUpdateLifetime: ulv,
			}
			c, err := cli.BuildAPIClient(opts)
			if err != nil {
				t.Fatalf("BuildAPIClient: %v", err)
			}
			if c == nil || c.Tokens == nil {
				t.Fatalf("client incomplete: %+v", c)
			}
		})
	}
}

func TestBuildAPIClientPlainWithLifetimeFlagsRejected(t *testing.T) {
	t.Setenv("KAS_LOGIN", "")
	t.Setenv("KAS_AUTHDATA", "")
	t.Setenv("KAS_AUTHTYPE", "")

	cases := []struct {
		name string
		opts *cli.RootOptions
	}{
		{"lifetime", &cli.RootOptions{
			Login: "w0000000", AuthData: "p", AuthType: "plain",
			SessionLifetime: 1800,
		}},
		{"update-lifetime", &cli.RootOptions{
			Login: "w0000000", AuthData: "p", AuthType: "plain",
			SessionUpdateLifetime: "Y",
		}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.opts.ConfigPath = filepath.Join(t.TempDir(), "missing-config.toml")
			_, err := cli.BuildAPIClient(tc.opts)
			if err == nil {
				t.Fatal("expected error")
			}
			var ee *cli.ExitError
			if !errors.As(err, &ee) || ee.Code != cli.ExitUserError {
				t.Errorf("got %v, want ExitUserError", err)
			}
		})
	}
}

func TestBuildAPIClientLifetimeRangeRejected(t *testing.T) {
	t.Setenv("KAS_LOGIN", "")
	t.Setenv("KAS_AUTHDATA", "")
	t.Setenv("KAS_AUTHTYPE", "")

	for _, lt := range []int{-1, 30001} {
		lt := lt
		t.Run("lifetime", func(t *testing.T) {
			t.Parallel()
			opts := &cli.RootOptions{
				ConfigPath:      filepath.Join(t.TempDir(), "missing-config.toml"),
				Login:           "w0000000",
				AuthData:        "secret-password",
				AuthType:        "session",
				SessionLifetime: lt,
			}
			_, err := cli.BuildAPIClient(opts)
			if err == nil {
				t.Fatalf("expected error for lifetime=%d", lt)
			}
		})
	}
}

func TestBuildAPIClientUpdateLifetimeInvalidRejected(t *testing.T) {
	t.Setenv("KAS_LOGIN", "")
	t.Setenv("KAS_AUTHDATA", "")
	t.Setenv("KAS_AUTHTYPE", "")

	opts := &cli.RootOptions{
		ConfigPath:            filepath.Join(t.TempDir(), "missing-config.toml"),
		Login:                 "w0000000",
		AuthData:              "secret-password",
		AuthType:              "session",
		SessionUpdateLifetime: "yes",
	}
	_, err := cli.BuildAPIClient(opts)
	if err == nil {
		t.Fatal("expected error for invalid update-lifetime")
	}
}
