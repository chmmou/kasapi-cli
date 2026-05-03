package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chmmou/kasapi-cli/internal/auth"
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
