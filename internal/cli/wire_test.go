package cli_test

import (
	"errors"
	"path/filepath"
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
