package cli_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"

	"github.com/chmmou/kasapi-cli/internal/cli"
	"github.com/chmmou/kasapi-cli/internal/config"
)

func newIO(in string, password string) (*bytes.Buffer, cli.ConfigIO) {
	out := &bytes.Buffer{}
	cio := cli.ConfigIO{
		In:           strings.NewReader(in),
		Out:          out,
		IsTTY:        func() bool { return true },
		ReadPassword: func() (string, error) { return password, nil },
	}
	return out, cio
}

func TestRunConfigInitCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "config.toml")
	out, cio := newIO("w0000000\nsession\n\n", "supersecret")

	if err := cli.RunConfigInit(path, "main", false, cio); err != nil {
		t.Fatalf("RunConfigInit: %v", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DefaultProfile != "main" {
		t.Errorf("DefaultProfile = %q, want main", cfg.DefaultProfile)
	}
	got := cfg.Profiles["main"]
	want := config.Profile{Login: "w0000000", AuthData: "supersecret", AuthType: "session"}
	if got != want {
		t.Errorf("profile = %+v, want %+v", got, want)
	}
	if !strings.Contains(out.String(), "Wrote profile") {
		t.Errorf("missing success line: %q", out.String())
	}
	if runtime.GOOS != "windows" {
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if mode := fi.Mode().Perm(); mode != 0o600 {
			t.Errorf("file mode = %o, want 0600", mode)
		}
	}
}

func TestRunConfigInitDefaultsAuthTypeSession(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	// empty auth_type line → default session
	_, cio := newIO("w0000000\n\n\n", "secret")

	if err := cli.RunConfigInit(path, "main", false, cio); err != nil {
		t.Fatalf("RunConfigInit: %v", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Profiles["main"].AuthType != "session" {
		t.Errorf("AuthType = %q, want session", cfg.Profiles["main"].AuthType)
	}
}

func TestRunConfigInitRejectsAuthTypeUntilValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	out, cio := newIO("w0000000\nbogus\nplain\n\n", "secret")

	if err := cli.RunConfigInit(path, "main", false, cio); err != nil {
		t.Fatalf("RunConfigInit: %v", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Profiles["main"].AuthType != "plain" {
		t.Errorf("AuthType = %q, want plain", cfg.Profiles["main"].AuthType)
	}
	if !strings.Contains(out.String(), "invalid auth_type") {
		t.Errorf("missing reprompt: %q", out.String())
	}
}

func TestRunConfigInitRefusesOverwriteWithoutForce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	pre := &config.Config{
		DefaultProfile: "main",
		Profiles: map[string]config.Profile{
			"main": {Login: "w0000000", AuthData: "old", AuthType: "session"},
		},
	}
	if err := pre.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	_, cio := newIO("w0000001\nplain\n", "newsecret")

	err := cli.RunConfigInit(path, "main", false, cio)
	if err == nil {
		t.Fatal("expected error for existing profile without --force")
	}
	var ee *cli.ExitError
	if !errors.As(err, &ee) || ee.Code != cli.ExitUserError {
		t.Errorf("expected ExitUserError, got %v", err)
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error %q does not mention already exists", err)
	}
	cfg, _ := config.Load(path)
	if cfg.Profiles["main"].AuthData != "old" {
		t.Errorf("file was overwritten: AuthData = %q", cfg.Profiles["main"].AuthData)
	}
}

func TestRunConfigInitForceOverwritesProfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	pre := &config.Config{
		DefaultProfile: "main",
		Profiles: map[string]config.Profile{
			"main": {Login: "w0000000", AuthData: "old", AuthType: "session"},
		},
	}
	if err := pre.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	_, cio := newIO("w0000001\nplain\n", "newsecret")

	if err := cli.RunConfigInit(path, "main", true, cio); err != nil {
		t.Fatalf("RunConfigInit: %v", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := config.Profile{Login: "w0000001", AuthData: "newsecret", AuthType: "plain"}
	if cfg.Profiles["main"] != want {
		t.Errorf("profile = %+v, want %+v", cfg.Profiles["main"], want)
	}
}

func TestRunConfigInitNonInteractive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	cio := cli.ConfigIO{
		In:           strings.NewReader(""),
		Out:          io.Discard,
		IsTTY:        func() bool { return false },
		ReadPassword: func() (string, error) { return "", nil },
	}
	err := cli.RunConfigInit(path, "main", false, cio)
	if err == nil {
		t.Fatal("expected non-TTY error")
	}
	var ee *cli.ExitError
	if !errors.As(err, &ee) || ee.Code != cli.ExitUserError {
		t.Errorf("expected ExitUserError, got %v", err)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Errorf("file should not have been created on non-TTY error: %v", statErr)
	}
}

func TestRunConfigInitRejectsEmptyLogin(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	_, cio := newIO("\n", "secret")
	err := cli.RunConfigInit(path, "main", false, cio)
	if err == nil {
		t.Fatal("expected error for empty login")
	}
	if !strings.Contains(err.Error(), "login is required") {
		t.Errorf("error %q does not mention login", err)
	}
}

func TestRunConfigInitRejectsEmptyAuthData(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	_, cio := newIO("w0000000\nsession\n", "")
	err := cli.RunConfigInit(path, "main", false, cio)
	if err == nil {
		t.Fatal("expected error for empty auth_data")
	}
	if !strings.Contains(err.Error(), "auth_data is required") {
		t.Errorf("error %q does not mention auth_data", err)
	}
}

func TestRunConfigInitDoesNotSetDefaultWhenDeclined(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	_, cio := newIO("w0000000\nsession\nn\n", "secret")

	if err := cli.RunConfigInit(path, "main", false, cio); err != nil {
		t.Fatalf("RunConfigInit: %v", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DefaultProfile != "" {
		t.Errorf("DefaultProfile = %q, want empty", cfg.DefaultProfile)
	}
}

func TestRunConfigInitKeepsExistingDefaultProfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	pre := &config.Config{
		DefaultProfile: "main",
		Profiles: map[string]config.Profile{
			"main": {Login: "w0000000", AuthData: "secret", AuthType: "session"},
		},
	}
	if err := pre.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// no default-profile prompt expected because one is already set
	_, cio := newIO("w0000001\nplain\n", "stagingsecret")

	if err := cli.RunConfigInit(path, "staging", false, cio); err != nil {
		t.Fatalf("RunConfigInit: %v", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DefaultProfile != "main" {
		t.Errorf("DefaultProfile = %q, want main (must not change)", cfg.DefaultProfile)
	}
	if _, ok := cfg.Profiles["staging"]; !ok {
		t.Errorf("staging profile missing")
	}
}

func TestRunConfigInitDoesNotLeakAuthDataInOutput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	out, cio := newIO("w0000000\nsession\n\n", "supersecretpwd")

	if err := cli.RunConfigInit(path, "main", false, cio); err != nil {
		t.Fatalf("RunConfigInit: %v", err)
	}
	if strings.Contains(out.String(), "supersecretpwd") {
		t.Errorf("auth_data leaked to stdout: %q", out.String())
	}
}

func TestConfigSaveReadback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	cfg := &config.Config{
		DefaultProfile: "main",
		Profiles: map[string]config.Profile{
			"main":    {Login: "w0000000", AuthData: "secret", AuthType: "session"},
			"staging": {Login: "w0000001", AuthData: "other", AuthType: "plain"},
		},
	}
	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	var got config.Config
	if _, err := toml.DecodeFile(path, &got); err != nil {
		t.Fatalf("DecodeFile: %v", err)
	}
	if got.DefaultProfile != "main" || len(got.Profiles) != 2 {
		t.Errorf("readback mismatch: %+v", got)
	}
}
