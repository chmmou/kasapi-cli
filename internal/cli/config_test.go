package cli_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/BurntSushi/toml"

	"github.com/chmmou/kasapi-cli/internal/cli"
	"github.com/chmmou/kasapi-cli/internal/config"
	"github.com/chmmou/kasapi-cli/internal/session"
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

// --- add-profile ----------------------------------------------------------

func TestRunConfigAddProfileCreatesEntry(t *testing.T) {
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
	_, cio := newIO("w0000001\nplain\n", "stagingsecret")

	if err := cli.RunConfigAddProfile(path, "staging", false, cio); err != nil {
		t.Fatalf("RunConfigAddProfile: %v", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DefaultProfile != "main" {
		t.Errorf("DefaultProfile = %q, want main (unchanged)", cfg.DefaultProfile)
	}
	want := config.Profile{Login: "w0000001", AuthData: "stagingsecret", AuthType: "plain"}
	if cfg.Profiles["staging"] != want {
		t.Errorf("staging = %+v, want %+v", cfg.Profiles["staging"], want)
	}
	if _, ok := cfg.Profiles["main"]; !ok {
		t.Errorf("main profile lost")
	}
}

func TestRunConfigAddProfileRefusesOverwriteWithoutForce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	pre := &config.Config{
		Profiles: map[string]config.Profile{
			"staging": {Login: "w0000001", AuthData: "old", AuthType: "session"},
		},
	}
	if err := pre.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	_, cio := newIO("w0000002\nplain\n", "new")
	err := cli.RunConfigAddProfile(path, "staging", false, cio)
	if err == nil {
		t.Fatal("expected error for existing profile without --force")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("err %q missing 'already exists'", err)
	}
	cfg, _ := config.Load(path)
	if cfg.Profiles["staging"].AuthData != "old" {
		t.Errorf("file overwritten without --force")
	}
}

func TestRunConfigAddProfileForceOverwrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	pre := &config.Config{
		Profiles: map[string]config.Profile{
			"staging": {Login: "w0000001", AuthData: "old", AuthType: "session"},
		},
	}
	if err := pre.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	_, cio := newIO("w0000002\nplain\n", "newpwd")
	if err := cli.RunConfigAddProfile(path, "staging", true, cio); err != nil {
		t.Fatalf("RunConfigAddProfile: %v", err)
	}
	cfg, _ := config.Load(path)
	want := config.Profile{Login: "w0000002", AuthData: "newpwd", AuthType: "plain"}
	if cfg.Profiles["staging"] != want {
		t.Errorf("staging = %+v, want %+v", cfg.Profiles["staging"], want)
	}
}

func TestRunConfigAddProfileSetsDefaultWhenEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	_, cio := newIO("w0000000\nsession\n", "secret")
	if err := cli.RunConfigAddProfile(path, "main", false, cio); err != nil {
		t.Fatalf("RunConfigAddProfile: %v", err)
	}
	cfg, _ := config.Load(path)
	if cfg.DefaultProfile != "main" {
		t.Errorf("DefaultProfile = %q, want main", cfg.DefaultProfile)
	}
}

// --- use-profile ----------------------------------------------------------

type revokeSpy struct {
	calls   int
	login   string
	token   string
	wantErr error
}

func (s *revokeSpy) fn() cli.RevokeFunc {
	return func(_ context.Context, login, token string) error {
		s.calls++
		s.login = login
		s.token = token
		return s.wantErr
	}
}

func newDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func twoProfileConfig(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "config.toml")
	pre := &config.Config{
		DefaultProfile: "main",
		Profiles: map[string]config.Profile{
			"main":    {Login: "w0000000", AuthData: "secret", AuthType: "session"},
			"staging": {Login: "w0000001", AuthData: "other", AuthType: "plain"},
		},
	}
	if err := pre.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	return path
}

func TestRunConfigUseProfileSwitchesDefault(t *testing.T) {
	dir := t.TempDir()
	cfgPath := twoProfileConfig(t, dir)
	storePath := filepath.Join(dir, "sessions.toml")
	store, err := session.New(storePath)
	if err != nil {
		t.Fatalf("session.New: %v", err)
	}
	spy := &revokeSpy{}
	out := &bytes.Buffer{}

	err = cli.RunConfigUseProfile(t.Context(), cfgPath, "staging", spy.fn(), store, newDiscardLogger(), out)
	if err != nil {
		t.Fatalf("RunConfigUseProfile: %v", err)
	}
	cfg, _ := config.Load(cfgPath)
	if cfg.DefaultProfile != "staging" {
		t.Errorf("DefaultProfile = %q, want staging", cfg.DefaultProfile)
	}
	if spy.calls != 0 {
		t.Errorf("revoke called %d times without cached session, want 0", spy.calls)
	}
}

func TestRunConfigUseProfileRejectsUnknown(t *testing.T) {
	dir := t.TempDir()
	cfgPath := twoProfileConfig(t, dir)
	store, _ := session.New(filepath.Join(dir, "sessions.toml"))
	spy := &revokeSpy{}

	err := cli.RunConfigUseProfile(t.Context(), cfgPath, "bogus", spy.fn(), store, newDiscardLogger(), io.Discard)
	if err == nil {
		t.Fatal("expected error for unknown profile")
	}
	if !strings.Contains(err.Error(), "not defined") {
		t.Errorf("err %q missing 'not defined'", err)
	}
	if spy.calls != 0 {
		t.Errorf("revoke called %d times for unknown profile, want 0", spy.calls)
	}
}

func TestRunConfigUseProfileNoOpForCurrentDefault(t *testing.T) {
	dir := t.TempDir()
	cfgPath := twoProfileConfig(t, dir)
	store, _ := session.New(filepath.Join(dir, "sessions.toml"))
	spy := &revokeSpy{}
	out := &bytes.Buffer{}

	err := cli.RunConfigUseProfile(t.Context(), cfgPath, "main", spy.fn(), store, newDiscardLogger(), out)
	if err != nil {
		t.Fatalf("RunConfigUseProfile: %v", err)
	}
	if spy.calls != 0 {
		t.Errorf("revoke called %d times for no-op switch, want 0", spy.calls)
	}
	if !strings.Contains(out.String(), "already the default") {
		t.Errorf("output missing 'already the default': %q", out.String())
	}
}

func TestRunConfigUseProfileInvalidatesCachedSession(t *testing.T) {
	dir := t.TempDir()
	cfgPath := twoProfileConfig(t, dir)
	storePath := filepath.Join(dir, "sessions.toml")
	store, err := session.New(storePath)
	if err != nil {
		t.Fatalf("session.New: %v", err)
	}
	pre := session.Entry{
		Token:     "01234567890abcdef0123456789abcdef0123456",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	if serr := store.Save(t.Context(), "w0000000", pre); serr != nil {
		t.Fatalf("Save: %v", serr)
	}
	spy := &revokeSpy{}

	err = cli.RunConfigUseProfile(t.Context(), cfgPath, "staging", spy.fn(), store, newDiscardLogger(), io.Discard)
	if err != nil {
		t.Fatalf("RunConfigUseProfile: %v", err)
	}
	if spy.calls != 1 {
		t.Errorf("revoke calls = %d, want 1", spy.calls)
	}
	if spy.login != "w0000000" || spy.token != pre.Token {
		t.Errorf("revoke called with (%q, %q), want (w0000000, %q)", spy.login, spy.token, pre.Token)
	}
	got, err := store.Load(t.Context(), "w0000000")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != nil {
		t.Errorf("session entry still present after switch: %+v", got)
	}
}

func TestRunConfigUseProfileTolerateServerError(t *testing.T) {
	dir := t.TempDir()
	cfgPath := twoProfileConfig(t, dir)
	storePath := filepath.Join(dir, "sessions.toml")
	store, _ := session.New(storePath)
	pre := session.Entry{
		Token:     "01234567890abcdef0123456789abcdef0123456",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	if err := store.Save(t.Context(), "w0000000", pre); err != nil {
		t.Fatalf("Save: %v", err)
	}
	spy := &revokeSpy{wantErr: errors.New("kas_session_invalid")}

	err := cli.RunConfigUseProfile(t.Context(), cfgPath, "staging", spy.fn(), store, newDiscardLogger(), io.Discard)
	if err != nil {
		t.Fatalf("switch must not fail on revoke error: %v", err)
	}
	cfg, _ := config.Load(cfgPath)
	if cfg.DefaultProfile != "staging" {
		t.Errorf("DefaultProfile = %q, want staging despite revoke error", cfg.DefaultProfile)
	}
	got, _ := store.Load(t.Context(), "w0000000")
	if got != nil {
		t.Errorf("session entry must be removed even on revoke error, got %+v", got)
	}
}

func TestRunConfigUseProfileSkipsRevokeWhenNoToken(t *testing.T) {
	dir := t.TempDir()
	cfgPath := twoProfileConfig(t, dir)
	store, _ := session.New(filepath.Join(dir, "sessions.toml"))
	spy := &revokeSpy{}

	err := cli.RunConfigUseProfile(t.Context(), cfgPath, "staging", spy.fn(), store, newDiscardLogger(), io.Discard)
	if err != nil {
		t.Fatalf("RunConfigUseProfile: %v", err)
	}
	if spy.calls != 0 {
		t.Errorf("revoke called %d times without cached token, want 0", spy.calls)
	}
}

// --- list-profiles --------------------------------------------------------

func TestRunConfigListProfilesMarksDefault(t *testing.T) {
	dir := t.TempDir()
	cfgPath := twoProfileConfig(t, dir)
	out := &bytes.Buffer{}
	if err := cli.RunConfigListProfiles(cfgPath, out); err != nil {
		t.Fatalf("RunConfigListProfiles: %v", err)
	}
	got := out.String()
	wantPrefix := "* main (session)\n  staging (plain)\n"
	if got != wantPrefix {
		t.Errorf("output = %q, want %q", got, wantPrefix)
	}
}

func TestRunConfigListProfilesNoConfigHint(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "missing.toml")
	out := &bytes.Buffer{}
	if err := cli.RunConfigListProfiles(cfgPath, out); err != nil {
		t.Fatalf("RunConfigListProfiles: %v", err)
	}
	if !strings.Contains(out.String(), "config init") {
		t.Errorf("missing first-run hint: %q", out.String())
	}
}

func TestRunConfigListProfilesDoesNotLeakAuthData(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	pre := &config.Config{
		DefaultProfile: "main",
		Profiles: map[string]config.Profile{
			"main": {Login: "w0000000", AuthData: "supersecretXYZ", AuthType: "session"},
		},
	}
	if err := pre.Save(cfgPath); err != nil {
		t.Fatalf("Save: %v", err)
	}
	out := &bytes.Buffer{}
	if err := cli.RunConfigListProfiles(cfgPath, out); err != nil {
		t.Fatalf("RunConfigListProfiles: %v", err)
	}
	if strings.Contains(out.String(), "supersecretXYZ") {
		t.Errorf("auth_data leaked: %q", out.String())
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
