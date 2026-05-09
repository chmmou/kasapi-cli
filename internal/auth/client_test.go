package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chmmou/kasapi-cli/internal/auth"
	"github.com/chmmou/kasapi-cli/internal/session"
	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/transport"
)

func newAuthClient(srv *httptest.Server, login, data string, typ soap.AuthType, opts auth.Options) *auth.Client {
	tr := transport.New()
	tr.HTTPClient = srv.Client()
	tr.MaxRetries = 0
	tr.Now = time.Now
	tr.Sleep = func(_ context.Context, _ time.Duration) error { return nil }
	c := auth.New(tr, login, data, typ, opts)
	c.Endpoint = srv.URL
	return c
}

func TestGetCredentialTokenSuccess(t *testing.T) {
	body := loadFixture(t, "session/add_session_response_success.xml")
	var got struct {
		body []byte
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.body = readAll(r)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	c := newAuthClient(srv, "w0000000", "password", soap.AuthPlain, auth.Options{Lifetime: 1800, OTP: "123456"})
	tok, err := c.GetCredentialToken(context.Background())
	if err != nil {
		t.Fatalf("GetCredentialToken: %v", err)
	}
	if tok != "01234567890abcdef0123456789abcdef0123456" {
		t.Errorf("token = %q", tok)
	}
	for _, want := range []string{`"kas_login":"w0000000"`, `"kas_auth_type":"plain"`, `"session_2fa":"123456"`} {
		if !strings.Contains(string(got.body), want) {
			t.Errorf("request missing %q: %s", want, got.body)
		}
	}
}

func TestGetCredentialTokenSurfacesTypedFault(t *testing.T) {
	body := loadFixture(t, "session/add_session_response_failed_otp_pin_incorrect.xml")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	c := newAuthClient(srv, "w0", "secret", soap.AuthPlain, auth.Options{OTP: "wrong"})
	_, err := c.GetCredentialToken(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	e := auth.AsError(err)
	if e == nil {
		t.Fatalf("expected *auth.Error, got %T", err)
	}
	if e.Code != "otp_pin_incorrect" {
		t.Errorf("Code = %q, want otp_pin_incorrect", e.Code)
	}
	if e.Login != "w0" {
		t.Errorf("Login = %q, want w0", e.Login)
	}
	if !auth.IsOTPPinIncorrect(err) {
		t.Error("IsOTPPinIncorrect = false, want true")
	}
}

func TestGetCredentialTokenRejectsEmptyCreds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("server should not be reached")
	}))
	defer srv.Close()
	c := newAuthClient(srv, "", "", "", auth.Options{})
	if _, err := c.GetCredentialToken(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestSessionTokenSourceCachesAndRefreshes(t *testing.T) {
	body := loadFixture(t, "session/add_session_response_success.xml")
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	c := newAuthClient(srv, "w0", "secret", soap.AuthPlain, auth.Options{})
	src := auth.NewSessionTokenSource(c)

	login, data, typ, err := src.Credentials(context.Background())
	if err != nil {
		t.Fatalf("Credentials: %v", err)
	}
	if login != "w0" || data == "" || typ != soap.AuthSession {
		t.Errorf("first call: login=%q data=%q typ=%q", login, data, typ)
	}
	if calls.Load() != 1 {
		t.Errorf("calls after first = %d, want 1", calls.Load())
	}

	// Second call must hit the cache, not the server.
	if _, _, _, err := src.Credentials(context.Background()); err != nil {
		t.Fatalf("second Credentials: %v", err)
	}
	if calls.Load() != 1 {
		t.Errorf("calls after cached second = %d, want 1", calls.Load())
	}

	// Invalidate forces a re-fetch.
	src.Invalidate()
	if _, _, _, err := src.Credentials(context.Background()); err != nil {
		t.Fatalf("third Credentials: %v", err)
	}
	if calls.Load() != 2 {
		t.Errorf("calls after invalidate = %d, want 2", calls.Load())
	}
}

func TestSessionTokenSourcePropagatesAuthError(t *testing.T) {
	body := loadFixture(t, "session/add_session_response_failed_otp_pin_incorrect.xml")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	src := auth.NewSessionTokenSource(newAuthClient(srv, "w0", "secret", soap.AuthPlain, auth.Options{}))
	_, _, _, err := src.Credentials(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !auth.IsOTPPinIncorrect(err) {
		t.Errorf("expected IsOTPPinIncorrect, got %v", err)
	}
}

func TestSessionTokenSourceLoadsFromStore(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("server should not be reached when a valid cached token exists")
	}))
	defer srv.Close()

	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	store := newStore(t, now)
	if err := store.Save("w0", session.Entry{
		Token:           "cached-tok",
		ExpiresAt:       now.Add(time.Hour),
		LifetimeSeconds: 3600,
	}); err != nil {
		t.Fatalf("seed Save: %v", err)
	}

	src := auth.NewSessionTokenSource(newAuthClient(srv, "w0", "secret", soap.AuthPlain, auth.Options{}))
	src.Store = store
	src.Now = func() time.Time { return now }

	login, data, typ, err := src.Credentials(context.Background())
	if err != nil {
		t.Fatalf("Credentials: %v", err)
	}
	if login != "w0" || data != "cached-tok" || typ != soap.AuthSession {
		t.Errorf("Credentials = (%q,%q,%q), want (w0, cached-tok, session)", login, data, typ)
	}
}

func TestSessionTokenSourceFetchesWhenStoredEntryExpired(t *testing.T) {
	body := loadFixture(t, "session/add_session_response_success.xml")
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	store := newStore(t, now)
	if err := store.Save("w0", session.Entry{
		Token:     "stale",
		ExpiresAt: now.Add(-time.Second),
	}); err != nil {
		t.Fatalf("seed Save: %v", err)
	}

	src := auth.NewSessionTokenSource(newAuthClient(srv, "w0", "secret", soap.AuthPlain, auth.Options{}))
	src.Store = store
	src.Lifetime = time.Hour
	src.Now = func() time.Time { return now }

	_, data, _, err := src.Credentials(context.Background())
	if err != nil {
		t.Fatalf("Credentials: %v", err)
	}
	if data == "stale" {
		t.Error("expected fresh token, got the stale cached one")
	}
	if calls.Load() != 1 {
		t.Errorf("KasAuth calls = %d, want 1", calls.Load())
	}
	saved, err := store.Load("w0")
	if err != nil || saved == nil {
		t.Fatalf("expected fresh entry persisted, got %v %v", saved, err)
	}
	if !saved.ExpiresAt.Equal(now.Add(time.Hour)) {
		t.Errorf("ExpiresAt = %v, want %v", saved.ExpiresAt, now.Add(time.Hour))
	}
}

func TestSessionTokenSourceInvalidateDeletesPersisted(t *testing.T) {
	body := loadFixture(t, "session/add_session_response_success.xml")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	store := newStore(t, now)
	src := auth.NewSessionTokenSource(newAuthClient(srv, "w0", "secret", soap.AuthPlain, auth.Options{}))
	src.Store = store
	src.Lifetime = time.Hour
	src.Now = func() time.Time { return now }

	if _, _, _, err := src.Credentials(context.Background()); err != nil {
		t.Fatalf("Credentials: %v", err)
	}
	if got, _ := store.Load("w0"); got == nil {
		t.Fatal("expected entry persisted after fresh fetch")
	}
	src.Invalidate()
	if got, _ := store.Load("w0"); got != nil {
		t.Errorf("expected entry deleted after Invalidate, got %+v", got)
	}
}

func TestSessionTokenSourceHeartbeatExtendsExpiry(t *testing.T) {
	body := loadFixture(t, "session/add_session_response_success.xml")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	tNow := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	store := newStore(t, tNow)
	src := auth.NewSessionTokenSource(newAuthClient(srv, "w0", "secret", soap.AuthPlain, auth.Options{}))
	src.Store = store
	src.Lifetime = time.Hour
	src.UpdateLifetime = true
	src.Now = func() time.Time { return tNow }

	if _, _, _, err := src.Credentials(context.Background()); err != nil {
		t.Fatalf("Credentials: %v", err)
	}
	first, _ := store.Load("w0")
	if first == nil {
		t.Fatal("expected initial entry")
	}

	tNow = tNow.Add(15 * time.Minute)
	store.Now = func() time.Time { return tNow }
	src.Heartbeat()

	got, _ := store.Load("w0")
	if got == nil {
		t.Fatal("expected entry after Heartbeat")
	}
	want := tNow.Add(time.Hour)
	if !got.ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt after Heartbeat = %v, want %v", got.ExpiresAt, want)
	}
}

func TestSessionTokenSourceAdoptsLifetimeFromCachedEntry(t *testing.T) {
	// Source created with no lifetime / update flags (e.g. a CLI run
	// without the KasAuth flags). It picks up a token persisted by an
	// earlier run that *did* enable session_update_lifetime, and must
	// honour those server-side properties on Heartbeat.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("server should not be reached when a valid cached token exists")
	}))
	defer srv.Close()

	tNow := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	store := newStore(t, tNow)
	if err := store.Save("w0", session.Entry{
		Token:           "cached",
		ExpiresAt:       tNow.Add(time.Hour),
		LifetimeSeconds: 3600,
		UpdateLifetime:  true,
	}); err != nil {
		t.Fatalf("seed Save: %v", err)
	}

	src := auth.NewSessionTokenSource(newAuthClient(srv, "w0", "secret", soap.AuthPlain, auth.Options{}))
	src.Store = store
	src.Now = func() time.Time { return tNow }

	if _, _, _, err := src.Credentials(context.Background()); err != nil {
		t.Fatalf("Credentials: %v", err)
	}

	tNow = tNow.Add(15 * time.Minute)
	store.Now = func() time.Time { return tNow }
	src.Heartbeat()

	got, _ := store.Load("w0")
	if got == nil {
		t.Fatal("expected entry after Heartbeat")
	}
	want := tNow.Add(time.Hour)
	if !got.ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt after adopted-Heartbeat = %v, want %v (lifetime should come from cached entry)", got.ExpiresAt, want)
	}
}

// TestSessionTokenSourceInvalidateRestoresConfiguredAfterAdopt covers
// the case the user reported as kas_session_invalid: an earlier run
// persisted a session with UpdateLifetime=N, the current run wires the
// source with --session-update-lifetime=Y, and the cached entry is
// adopted (overwriting the wired flags). When the server later rejects
// the adopted token, Invalidate must restore the wired flags so the
// fresh session is persisted with the user's current intent.
func TestSessionTokenSourceInvalidateRestoresConfiguredAfterAdopt(t *testing.T) {
	body := loadFixture(t, "session/add_session_response_success.xml")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	tNow := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	store := newStore(t, tNow)
	if err := store.Save("w0", session.Entry{
		Token:           "stale-but-locally-valid",
		ExpiresAt:       tNow.Add(30 * time.Minute),
		LifetimeSeconds: 1800,
		UpdateLifetime:  false,
	}); err != nil {
		t.Fatalf("seed Save: %v", err)
	}

	src := auth.NewSessionTokenSource(newAuthClient(srv, "w0", "secret", soap.AuthPlain, auth.Options{}))
	src.Store = store
	src.Lifetime = 2 * time.Hour
	src.UpdateLifetime = true
	src.Now = func() time.Time { return tNow }

	// First Credentials call adopts the cached entry's lifetime / flag.
	if _, _, _, err := src.Credentials(context.Background()); err != nil {
		t.Fatalf("Credentials: %v", err)
	}

	// Simulate the server-side kas_session_invalid by invalidating, then
	// fetching fresh credentials. The new entry must be persisted with
	// the wired Lifetime+UpdateLifetime, not the adopted ones.
	src.Invalidate()
	if _, _, _, err := src.Credentials(context.Background()); err != nil {
		t.Fatalf("Credentials after invalidate: %v", err)
	}

	got, _ := store.Load("w0")
	if got == nil {
		t.Fatal("expected fresh entry persisted after invalidate")
	}
	if got.LifetimeSeconds != int(2*time.Hour/time.Second) {
		t.Errorf("LifetimeSeconds = %d, want %d (wired value)", got.LifetimeSeconds, int(2*time.Hour/time.Second))
	}
	if !got.UpdateLifetime {
		t.Errorf("UpdateLifetime = false, want true (wired value)")
	}
}

func TestSessionTokenSourceHeartbeatNoopWithoutUpdateLifetime(t *testing.T) {
	body := loadFixture(t, "session/add_session_response_success.xml")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	tNow := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	store := newStore(t, tNow)
	src := auth.NewSessionTokenSource(newAuthClient(srv, "w0", "secret", soap.AuthPlain, auth.Options{}))
	src.Store = store
	src.Lifetime = time.Hour
	src.Now = func() time.Time { return tNow }

	if _, _, _, err := src.Credentials(context.Background()); err != nil {
		t.Fatalf("Credentials: %v", err)
	}
	original, _ := store.Load("w0")
	if original == nil {
		t.Fatal("expected initial entry")
	}

	tNow = tNow.Add(15 * time.Minute)
	src.Heartbeat()

	got, _ := store.Load("w0")
	if got == nil || !got.ExpiresAt.Equal(original.ExpiresAt) {
		t.Errorf("ExpiresAt changed without UpdateLifetime: %+v", got)
	}
}

func newStore(t *testing.T, now time.Time) *session.Store {
	t.Helper()
	s, err := session.New(filepath.Join(t.TempDir(), "sessions.toml"))
	if err != nil {
		t.Fatalf("session.New: %v", err)
	}
	s.Now = func() time.Time { return now }
	return s
}

func readAll(r *http.Request) []byte {
	defer func() { _ = r.Body.Close() }()
	out := make([]byte, 0, 1024)
	buf := make([]byte, 1024)
	for {
		n, err := r.Body.Read(buf)
		out = append(out, buf[:n]...)
		if err != nil {
			break
		}
	}
	return out
}
