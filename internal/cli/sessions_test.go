package cli_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chmmou/kasapi-cli/internal/api"
	"github.com/chmmou/kasapi-cli/internal/cli"
	"github.com/chmmou/kasapi-cli/internal/session"
)

const fixtureToken = "01234567890abcdef0123456789abcdef0123456"

// clearKASEnv makes resolveCreds hermetic regardless of the developer's
// shell: the KAS_* env vars are consulted by config.EnvFromOS and would
// otherwise leak into the profile resolution.
func clearKASEnv(t *testing.T) {
	t.Helper()
	t.Setenv("KAS_LOGIN", "")
	t.Setenv("KAS_AUTHDATA", "")
	t.Setenv("KAS_AUTHTYPE", "")
}

func storeWithSession(t *testing.T, dir, login, token string) *session.Store {
	t.Helper()
	store, err := session.New(filepath.Join(dir, "sessions.toml"))
	if err != nil {
		t.Fatalf("session.New: %v", err)
	}
	if err := store.Save(t.Context(), login, session.Entry{
		Token:     token,
		ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}
	return store
}

func TestRunSessionsDeleteRevokesCachedSession(t *testing.T) {
	clearKASEnv(t)
	dir := t.TempDir()
	cfgPath := twoProfileConfig(t, dir)
	store := storeWithSession(t, dir, "w0000000", fixtureToken)
	spy := &revokeSpy{}
	out := &bytes.Buffer{}

	opts := &cli.RootOptions{ConfigPath: cfgPath}
	if err := cli.RunSessionsDelete(t.Context(), opts, spy.fn(), store, newDiscardLogger(), out); err != nil {
		t.Fatalf("RunSessionsDelete: %v", err)
	}
	if spy.calls != 1 {
		t.Fatalf("revoke called %d times, want 1", spy.calls)
	}
	if spy.login != "w0000000" || spy.token != fixtureToken {
		t.Errorf("revoke(login=%q token=%q), want (w0000000, %s)", spy.login, spy.token, fixtureToken)
	}
	if e, _ := store.Load(t.Context(), "w0000000"); e != nil {
		t.Errorf("local cache entry still present after delete: %+v", e)
	}
	if !strings.Contains(out.String(), "Deleted server-side session") {
		t.Errorf("output missing success line: %q", out.String())
	}
}

func TestRunSessionsDeleteNoCachedSession(t *testing.T) {
	clearKASEnv(t)
	dir := t.TempDir()
	cfgPath := twoProfileConfig(t, dir)
	store, _ := session.New(filepath.Join(dir, "sessions.toml"))
	spy := &revokeSpy{}
	out := &bytes.Buffer{}

	opts := &cli.RootOptions{ConfigPath: cfgPath}
	if err := cli.RunSessionsDelete(t.Context(), opts, spy.fn(), store, newDiscardLogger(), out); err != nil {
		t.Fatalf("RunSessionsDelete: %v", err)
	}
	if spy.calls != 0 {
		t.Errorf("revoke called %d times without cached session, want 0", spy.calls)
	}
	if !strings.Contains(out.String(), "nothing to revoke") {
		t.Errorf("output missing 'nothing to revoke': %q", out.String())
	}
}

func TestRunSessionsDeletePlainProfile(t *testing.T) {
	clearKASEnv(t)
	dir := t.TempDir()
	cfgPath := twoProfileConfig(t, dir)
	store, _ := session.New(filepath.Join(dir, "sessions.toml"))
	spy := &revokeSpy{}
	out := &bytes.Buffer{}

	// staging is auth_type=plain in twoProfileConfig.
	opts := &cli.RootOptions{ConfigPath: cfgPath, Profile: "staging"}
	if err := cli.RunSessionsDelete(t.Context(), opts, spy.fn(), store, newDiscardLogger(), out); err != nil {
		t.Fatalf("RunSessionsDelete: %v", err)
	}
	if spy.calls != 0 {
		t.Errorf("revoke called %d times for plain profile, want 0", spy.calls)
	}
	if !strings.Contains(out.String(), "auth_type=plain") {
		t.Errorf("output missing plain-profile note: %q", out.String())
	}
}

func TestRunSessionsDeleteToleratesUnknownSession(t *testing.T) {
	clearKASEnv(t)
	dir := t.TempDir()
	cfgPath := twoProfileConfig(t, dir)
	store := storeWithSession(t, dir, "w0000000", fixtureToken)
	spy := &revokeSpy{wantErr: &api.Error{Code: api.CodeUnknownSession, Action: "delete_session"}}
	out := &bytes.Buffer{}

	opts := &cli.RootOptions{ConfigPath: cfgPath}
	if err := cli.RunSessionsDelete(t.Context(), opts, spy.fn(), store, newDiscardLogger(), out); err != nil {
		t.Fatalf("RunSessionsDelete returned error for already-invalid session, want nil: %v", err)
	}
	if spy.calls != 1 {
		t.Errorf("revoke called %d times, want 1", spy.calls)
	}
	if e, _ := store.Load(t.Context(), "w0000000"); e != nil {
		t.Errorf("local cache not cleared after unknown_session: %+v", e)
	}
	if !strings.Contains(out.String(), "already invalid") {
		t.Errorf("output missing idempotent note: %q", out.String())
	}
}

func TestRunSessionsDeleteSurfacesTransportError(t *testing.T) {
	clearKASEnv(t)
	dir := t.TempDir()
	cfgPath := twoProfileConfig(t, dir)
	store := storeWithSession(t, dir, "w0000000", fixtureToken)
	spy := &revokeSpy{wantErr: errors.New("boom")}
	out := &bytes.Buffer{}

	opts := &cli.RootOptions{ConfigPath: cfgPath}
	err := cli.RunSessionsDelete(t.Context(), opts, spy.fn(), store, newDiscardLogger(), out)
	if err == nil {
		t.Fatal("RunSessionsDelete err = nil, want a surfaced transport error")
	}
	if spy.calls != 1 {
		t.Errorf("revoke called %d times, want 1", spy.calls)
	}
	// Local cache is authoritative: it is cleared even though the
	// server-side revoke failed.
	if e, _ := store.Load(t.Context(), "w0000000"); e != nil {
		t.Errorf("local cache not cleared after transport error: %+v", e)
	}
}

func TestRunSessionsDeleteNoConfigHint(t *testing.T) {
	clearKASEnv(t)
	dir := t.TempDir()
	store, _ := session.New(filepath.Join(dir, "sessions.toml"))
	spy := &revokeSpy{}

	opts := &cli.RootOptions{ConfigPath: filepath.Join(dir, "missing.toml")}
	err := cli.RunSessionsDelete(t.Context(), opts, spy.fn(), store, newDiscardLogger(), io.Discard)
	if err == nil {
		t.Fatal("expected error when no config file exists")
	}
	if !strings.Contains(err.Error(), "config init") {
		t.Errorf("err %q missing first-run `config init` hint", err)
	}
	if spy.calls != 0 {
		t.Errorf("revoke called %d times without config, want 0", spy.calls)
	}
}

// --- integration against testdata fixtures (real soap.Decode pipeline) ---

func TestSessionsDeleteCallsDeleteSessionAction(t *testing.T) {
	clearKASEnv(t)
	dir := t.TempDir()
	cfgPath := twoProfileConfig(t, dir)
	store := storeWithSession(t, dir, "w0000000", fixtureToken)

	var captured []byte
	srv := newRevokeServer(t, "../../testdata/session/delete_session_response_success.xml", &captured)
	defer srv.Close()

	logger := newDiscardLogger()
	revoke := func(ctx context.Context, login, token string) error {
		return cli.RevokeSession(ctx, login, token, srv.URL, logger)
	}
	opts := &cli.RootOptions{ConfigPath: cfgPath}
	out := &bytes.Buffer{}
	if err := cli.RunSessionsDelete(t.Context(), opts, revoke, store, logger, out); err != nil {
		t.Fatalf("RunSessionsDelete: %v", err)
	}
	body := string(captured)
	for _, want := range []string{
		`"kas_action":"delete_session"`,
		`"kas_auth_type":"session"`,
		`"kas_login":"w0000000"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("request body missing %s: %s", want, body)
		}
	}
}

func TestSessionsDeleteToleratesUnknownSessionFault(t *testing.T) {
	clearKASEnv(t)
	dir := t.TempDir()
	cfgPath := twoProfileConfig(t, dir)
	store := storeWithSession(t, dir, "w0000000", "expiredtoken")

	srv := newRevokeServer(t, "../../testdata/session/delete_session_response_failed_unknown_session.xml", nil)
	defer srv.Close()

	logger := newDiscardLogger()
	revoke := func(ctx context.Context, login, token string) error {
		return cli.RevokeSession(ctx, login, token, srv.URL, logger)
	}
	opts := &cli.RootOptions{ConfigPath: cfgPath}
	out := &bytes.Buffer{}
	if err := cli.RunSessionsDelete(t.Context(), opts, revoke, store, logger, out); err != nil {
		t.Fatalf("RunSessionsDelete returned error for unknown_session fault, want idempotent nil: %v", err)
	}
	if e, _ := store.Load(t.Context(), "w0000000"); e != nil {
		t.Errorf("local cache not cleared after unknown_session fault: %+v", e)
	}
	if !strings.Contains(out.String(), "already invalid") {
		t.Errorf("output missing idempotent note: %q", out.String())
	}
}
