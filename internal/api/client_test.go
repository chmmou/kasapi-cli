package api_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chmmou/kasapi-cli/internal/api"
	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/testutil"
	"github.com/chmmou/kasapi-cli/internal/transport"
)

func loadFixture(t *testing.T, rel string) []byte {
	t.Helper()
	//nolint:gosec // G304: test fixture loader, path is rooted at testutil.RepoRoot(t).
	b, err := os.ReadFile(filepath.Join(testutil.RepoRoot(t), "testdata", rel))
	if err != nil {
		t.Fatalf("load %s: %v", rel, err)
	}
	return b
}

func newAPIClient(srv *httptest.Server, ts api.TokenSource) *api.Client {
	tr := transport.New()
	tr.HTTPClient = srv.Client()
	tr.MaxRetries = 0
	tr.Now = time.Now
	tr.Sleep = func(_ context.Context, _ time.Duration) error { return nil }
	c := api.New(tr, ts)
	c.Endpoint = srv.URL
	return c
}

func staticTokens() *api.StaticTokenSource {
	return &api.StaticTokenSource{
		Login:    "w0000000",
		AuthData: "secret",
		AuthType: soap.AuthSession,
	}
}

func TestCallSuccess(t *testing.T) {
	body := loadFixture(t, "account/get_accounts_response_success.xml")
	var got struct {
		method string
		body   []byte
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.method = r.Method
		got.body, _ = io.ReadAll(r.Body)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	c := newAPIClient(srv, staticTokens())
	resp, err := c.Call(context.Background(), "get_accounts", nil)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if resp.Body.ReturnString != "TRUE" {
		t.Errorf("ReturnString = %q, want TRUE", resp.Body.ReturnString)
	}
	if got.method != http.MethodPost {
		t.Errorf("method = %q, want POST", got.method)
	}
	if !strings.Contains(string(got.body), `"kas_action":"get_accounts"`) {
		t.Errorf("encoded request missing kas_action: %s", got.body)
	}
}

func TestCallFeedsKasFloodDelayToTransport(t *testing.T) {
	body := loadFixture(t, "account/get_accounts_response_success.xml")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	tr := transport.New()
	tr.HTTPClient = srv.Client()
	tr.MaxRetries = 0
	tr.Now = time.Now
	var slept atomic.Int64
	tr.Sleep = func(_ context.Context, d time.Duration) error {
		slept.Add(int64(d))
		return nil
	}
	c := api.New(tr, staticTokens())
	c.Endpoint = srv.URL

	if _, err := c.Call(context.Background(), "get_accounts", nil); err != nil {
		t.Fatalf("Call 1: %v", err)
	}
	if slept.Load() != 0 {
		t.Errorf("first call slept %v, want 0", time.Duration(slept.Load()))
	}
	if _, err := c.Call(context.Background(), "get_accounts", nil); err != nil {
		t.Fatalf("Call 2: %v", err)
	}
	if got := time.Duration(slept.Load()); got < 400*time.Millisecond || got > 600*time.Millisecond {
		t.Errorf("second call gate slept %v, want ~500ms (KasFloodDelay=0.5)", got)
	}
}

func TestCallReturnsTypedFault(t *testing.T) {
	body := loadFixture(t, "account/add_account_response_failed_max_account_reached.xml")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	c := newAPIClient(srv, staticTokens())
	_, err := c.Call(context.Background(), "add_account", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr := api.AsError(err)
	if apiErr == nil {
		t.Fatalf("expected *api.Error, got %T: %v", err, err)
	}
	if apiErr.Code != "max_account_reached" {
		t.Errorf("Code = %q, want max_account_reached", apiErr.Code)
	}
	if apiErr.Action != "add_account" {
		t.Errorf("Action = %q, want add_account", apiErr.Action)
	}
	if !api.IsMaxReached(err) {
		t.Error("IsMaxReached(err) = false, want true")
	}
	// Underlying SOAP fault must still be reachable via errors.As.
	var fe *soap.FaultError
	if !errors.As(err, &fe) {
		t.Error("errors.As to *soap.FaultError failed")
	}
}

// TestCallReturnsTypedFaultOnHTTP500 pins the end-to-end contract
// behind the transport 5xx fault pass-through: a KAS fault delivered
// with HTTP 500 (PHP SOAP servers do this) must reach the decoder and
// surface as the same typed *api.Error a 200-wrapped fault produces,
// instead of being burned in blind transport retries.
func TestCallReturnsTypedFaultOnHTTP500(t *testing.T) {
	body := loadFixture(t, "account/add_account_response_failed_max_account_reached.xml")
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	tr := transport.New()
	tr.HTTPClient = srv.Client()
	tr.MaxRetries = 2
	tr.Now = time.Now
	tr.Sleep = func(_ context.Context, _ time.Duration) error { return nil }
	c := api.New(tr, staticTokens())
	c.Endpoint = srv.URL

	_, err := c.Call(context.Background(), "add_account", nil)
	apiErr := api.AsError(err)
	if apiErr == nil {
		t.Fatalf("expected *api.Error, got %T: %v", err, err)
	}
	if apiErr.Code != "max_account_reached" {
		t.Errorf("Code = %q, want max_account_reached", apiErr.Code)
	}
	if calls.Load() != 1 {
		t.Errorf("calls = %d, want 1 (500-wrapped fault must not be retried)", calls.Load())
	}
}

func TestCallRetriesOnAuthFailure(t *testing.T) {
	authBody := loadFixture(t, "response_failed_no_auth.xml")
	okBody := loadFixture(t, "account/get_accounts_response_success.xml")
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			_, _ = w.Write(authBody)
			return
		}
		_, _ = w.Write(okBody)
	}))
	defer srv.Close()

	ts := &countingTokens{login: "w0", data: "tok-1", typ: soap.AuthSession, refresh: "tok-2"}
	c := newAPIClient(srv, ts)
	resp, err := c.Call(context.Background(), "get_accounts", nil)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if resp.Body.ReturnString != "TRUE" {
		t.Errorf("ReturnString = %q, want TRUE", resp.Body.ReturnString)
	}
	if calls.Load() != 2 {
		t.Errorf("server calls = %d, want 2", calls.Load())
	}
	if ts.invalidations != 1 {
		t.Errorf("invalidations = %d, want 1", ts.invalidations)
	}
	if ts.lastData != "tok-2" {
		t.Errorf("retry used data = %q, want tok-2", ts.lastData)
	}
}

// TestCallRetriesOnSessionInvalid is the kas_session_invalid analogue
// of TestCallRetriesOnAuthFailure: it pins the integration so a future
// edit that drops kas_session_invalid from IsAuthFailure surfaces here
// rather than only in TestPredicateClassesByCode.
func TestCallRetriesOnSessionInvalid(t *testing.T) {
	authBody := loadFixture(t, "response_failed_kas_session_invalid.xml")
	okBody := loadFixture(t, "account/get_accounts_response_success.xml")
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			_, _ = w.Write(authBody)
			return
		}
		_, _ = w.Write(okBody)
	}))
	defer srv.Close()

	ts := &countingTokens{login: "w0", data: "tok-1", typ: soap.AuthSession, refresh: "tok-2"}
	c := newAPIClient(srv, ts)
	if _, err := c.Call(context.Background(), "get_accounts", nil); err != nil {
		t.Fatalf("Call: %v", err)
	}
	if calls.Load() != 2 {
		t.Errorf("server calls = %d, want 2", calls.Load())
	}
	if ts.invalidations != 1 {
		t.Errorf("invalidations = %d, want 1", ts.invalidations)
	}
	if ts.lastData != "tok-2" {
		t.Errorf("retry used data = %q, want tok-2", ts.lastData)
	}
}

// A static credential cannot refresh itself, so an auth failure with a
// StaticTokenSource is terminal: retrying with identical credentials
// would only double the failing request against the flood gate.
func TestCallNoRetryOnAuthFailureWithStaticTokens(t *testing.T) {
	body := loadFixture(t, "response_failed_no_auth.xml")
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	c := newAPIClient(srv, staticTokens())
	if _, err := c.Call(context.Background(), "get_accounts", nil); err == nil {
		t.Fatal("expected auth-failure error")
	}
	if calls.Load() != 1 {
		t.Errorf("server calls = %d, want 1 (no retry with non-refreshable credentials)", calls.Load())
	}
}

func TestCallNoRetryOnNonAuthFault(t *testing.T) {
	body := loadFixture(t, "account/add_account_response_failed_max_account_reached.xml")
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	c := newAPIClient(srv, staticTokens())
	if _, err := c.Call(context.Background(), "add_account", nil); err == nil {
		t.Fatal("expected error")
	}
	if calls.Load() != 1 {
		t.Errorf("server calls = %d, want 1 (no retry on max_account_reached)", calls.Load())
	}
}

func TestCallRejectsEmptyAction(t *testing.T) {
	c := newAPIClient(httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("server should not be reached")
	})), staticTokens())
	if _, err := c.Call(context.Background(), "", nil); err == nil {
		t.Fatal("expected error on empty action")
	}
}

func TestCallHeartbeatsTokenSourceOnSuccess(t *testing.T) {
	body := loadFixture(t, "account/get_accounts_response_success.xml")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	ts := &beatingTokens{login: "w0", data: "tok", typ: soap.AuthSession}
	c := newAPIClient(srv, ts)
	if _, err := c.Call(context.Background(), "get_accounts", nil); err != nil {
		t.Fatalf("Call: %v", err)
	}
	if ts.heartbeats != 1 {
		t.Errorf("heartbeats = %d, want 1", ts.heartbeats)
	}
}

func TestCallNoHeartbeatOnFailure(t *testing.T) {
	body := loadFixture(t, "account/add_account_response_failed_max_account_reached.xml")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	ts := &beatingTokens{login: "w0", data: "tok", typ: soap.AuthSession}
	c := newAPIClient(srv, ts)
	if _, err := c.Call(context.Background(), "add_account", nil); err == nil {
		t.Fatal("expected error")
	}
	if ts.heartbeats != 0 {
		t.Errorf("heartbeats = %d, want 0 on fault", ts.heartbeats)
	}
}

// countingTokens is a TokenSource that swaps AuthData on Invalidate so
// tests can confirm the retry pulled fresh credentials.
type countingTokens struct {
	login         string
	data          string
	typ           soap.AuthType
	refresh       string
	lastData      string
	invalidations int
}

func (c *countingTokens) Credentials(_ context.Context) (string, string, soap.AuthType, error) {
	c.lastData = c.data
	return c.login, c.data, c.typ, nil
}

func (c *countingTokens) Invalidate() bool {
	c.invalidations++
	c.data = c.refresh
	return true
}

// beatingTokens implements TokenSource + Heartbeater to verify the
// post-success hook in Client.Call.
type beatingTokens struct {
	login      string
	data       string
	typ        soap.AuthType
	heartbeats int
}

func (b *beatingTokens) Credentials(_ context.Context) (string, string, soap.AuthType, error) {
	return b.login, b.data, b.typ, nil
}
func (b *beatingTokens) Invalidate() bool            { return true }
func (b *beatingTokens) Heartbeat(_ context.Context) { b.heartbeats++ }
