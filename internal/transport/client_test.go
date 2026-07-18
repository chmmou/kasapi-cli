package transport_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chmmou/kasapi-cli/internal/transport"
)

const sampleEnvelope = `<?xml version="1.0"?><soapenv:Envelope/>`

// fakeClock returns a Now func backed by a manually advanced timestamp
// and a Sleep func that records every requested wait.
type fakeClock struct {
	mu   sync.Mutex
	now  time.Time
	naps []time.Duration
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
}

func (f *fakeClock) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

func (f *fakeClock) Sleep(_ context.Context, d time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.naps = append(f.naps, d)
	f.now = f.now.Add(d)
	return nil
}

func (f *fakeClock) Naps() []time.Duration {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]time.Duration, len(f.naps))
	copy(out, f.naps)
	return out
}

func newClient(srv *httptest.Server, fc *fakeClock) *transport.Client {
	c := transport.New()
	c.HTTPClient = srv.Client()
	c.Now = fc.Now
	c.Sleep = fc.Sleep
	return c
}

func TestDoSuccess(t *testing.T) {
	var got struct {
		method      string
		contentType string
		userAgent   string
		body        []byte
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.method = r.Method
		got.contentType = r.Header.Get("Content-Type")
		got.userAgent = r.Header.Get("User-Agent")
		got.body, _ = io.ReadAll(r.Body)
		_, _ = io.WriteString(w, "<ok/>")
	}))
	defer srv.Close()

	c := newClient(srv, newFakeClock())
	resp, err := c.Do(context.Background(), srv.URL, []byte(sampleEnvelope))
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if string(resp) != "<ok/>" {
		t.Errorf("body = %q, want <ok/>", resp)
	}
	if got.method != http.MethodPost {
		t.Errorf("method = %q, want POST", got.method)
	}
	if got.contentType != "text/xml; charset=utf-8" {
		t.Errorf("content-type = %q", got.contentType)
	}
	if got.userAgent == "" || got.userAgent[:11] != "kasapi-cli/" {
		t.Errorf("user-agent = %q, want kasapi-cli/...", got.userAgent)
	}
	if string(got.body) != sampleEnvelope {
		t.Errorf("server received %q, want %q", got.body, sampleEnvelope)
	}
}

// Regression for kasserver responses returned with Content-Encoding:
// gzip. net/http only decompresses transparently when the caller has
// *not* set Accept-Encoding; an explicit Accept-Encoding: gzip header
// disables that and leaks raw gzip bytes into the XML decoder
// ("invalid character entity &…").
func TestDoTransparentlyDecompressesGzip(t *testing.T) {
	const payload = `<?xml version="1.0"?><soapenv:Envelope><Body>ok</Body></soapenv:Envelope>`

	var sentAcceptEncoding string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sentAcceptEncoding = r.Header.Get("Accept-Encoding")
		// Only gzip-encode if the client claims to accept gzip — net/http
		// adds it automatically when the caller did not set the header.
		if !strings.Contains(sentAcceptEncoding, "gzip") {
			_, _ = io.WriteString(w, payload)
			return
		}
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		_, _ = gz.Write([]byte(payload))
		_ = gz.Close()
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Content-Type", "text/xml")
		_, _ = w.Write(buf.Bytes())
	}))
	defer srv.Close()

	c := newClient(srv, newFakeClock())
	resp, err := c.Do(context.Background(), srv.URL, []byte(sampleEnvelope))
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if string(resp) != payload {
		t.Errorf("body = %q, want %q", resp, payload)
	}
	if !strings.Contains(sentAcceptEncoding, "gzip") {
		t.Errorf("Accept-Encoding = %q, want it to contain gzip (net/http auto-adds it)", sentAcceptEncoding)
	}
}

func TestDoRetriesOn5xx(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n < 3 {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		_, _ = io.WriteString(w, "<ok/>")
	}))
	defer srv.Close()

	fc := newFakeClock()
	c := newClient(srv, fc)
	_, err := c.Do(context.Background(), srv.URL, []byte(sampleEnvelope))
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if calls.Load() != 3 {
		t.Errorf("calls = %d, want 3", calls.Load())
	}
	naps := fc.Naps()
	if len(naps) != 2 {
		t.Fatalf("naps = %v, want 2 backoffs", naps)
	}
	if naps[0] != 500*time.Millisecond || naps[1] != time.Second {
		t.Errorf("naps = %v, want [500ms 1s]", naps)
	}
}

func TestDoPassesThroughSoapFaultOn5xx(t *testing.T) {
	const faultBody = `<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/">` +
		`<SOAP-ENV:Body><SOAP-ENV:Fault><faultcode>SOAP-ENV:Server</faultcode>` +
		`<faultstring>flood_protection</faultstring></SOAP-ENV:Fault></SOAP-ENV:Body></SOAP-ENV:Envelope>`
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, faultBody)
	}))
	defer srv.Close()

	c := newClient(srv, newFakeClock())
	body, err := c.Do(context.Background(), srv.URL, []byte(sampleEnvelope))
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if string(body) != faultBody {
		t.Errorf("body = %q, want the fault body passed through", body)
	}
	if calls.Load() != 1 {
		t.Errorf("calls = %d, want 1 (fault body must not be retried)", calls.Load())
	}
}

func TestDoPassesThroughDefaultNamespaceFaultOn5xx(t *testing.T) {
	const faultBody = `<Envelope xmlns="http://schemas.xmlsoap.org/soap/envelope/">` +
		`<Body><Fault><faultcode>Server</faultcode>` +
		`<faultstring>flood_protection</faultstring></Fault></Body></Envelope>`
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, faultBody)
	}))
	defer srv.Close()

	c := newClient(srv, newFakeClock())
	body, err := c.Do(context.Background(), srv.URL, []byte(sampleEnvelope))
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if string(body) != faultBody {
		t.Errorf("body = %q, want the prefix-less fault body passed through", body)
	}
	if calls.Load() != 1 {
		t.Errorf("calls = %d, want 1 (fault body must not be retried)", calls.Load())
	}
}

func TestDoStopsRetryAfterMax(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newClient(srv, newFakeClock())
	c.MaxRetries = 2
	_, err := c.Do(context.Background(), srv.URL, []byte(sampleEnvelope))
	if err == nil {
		t.Fatal("expected error after max retries")
	}
	if calls.Load() != 3 {
		t.Errorf("calls = %d, want 3 (1 + 2 retries)", calls.Load())
	}
}

func TestDoNoRetryOn4xx(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	c := newClient(srv, newFakeClock())
	_, err := c.Do(context.Background(), srv.URL, []byte(sampleEnvelope))
	if err == nil {
		t.Fatal("expected error on 4xx")
	}
	if calls.Load() != 1 {
		t.Errorf("calls = %d, want 1 (no retry)", calls.Load())
	}
}

func TestDoRetriesOnNetworkError(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Fatal("ResponseWriter does not support hijack")
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				t.Fatalf("hijack: %v", err)
			}
			_ = conn.Close()
			return
		}
		_, _ = io.WriteString(w, "<ok/>")
	}))
	defer srv.Close()

	c := newClient(srv, newFakeClock())
	_, err := c.Do(context.Background(), srv.URL, []byte(sampleEnvelope))
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if calls.Load() < 2 {
		t.Errorf("calls = %d, want >=2", calls.Load())
	}
}

func TestDoContextCancelDuringBackoff(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := transport.New()
	c.HTTPClient = srv.Client()
	c.Sleep = func(ctx context.Context, _ time.Duration) error {
		return context.Canceled
	}
	c.Now = time.Now
	_, err := c.Do(context.Background(), srv.URL, []byte(sampleEnvelope))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if !strings.Contains(err.Error(), "retry backoff interrupted") {
		t.Errorf("err = %q, want it to carry the retry-backoff phase context", err)
	}
}

func TestDoContextCancelDuringGateWait(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "<ok/>")
	}))
	defer srv.Close()

	c := transport.New()
	c.HTTPClient = srv.Client()
	c.Now = time.Now
	c.Sleep = func(_ context.Context, _ time.Duration) error {
		return context.Canceled
	}
	c.RecordDelay(500 * time.Millisecond)

	_, err := c.Do(context.Background(), srv.URL, []byte(sampleEnvelope))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want errors.Is context.Canceled", err)
	}
	if !strings.Contains(err.Error(), "flood-gate wait interrupted") {
		t.Errorf("err = %q, want it to carry the flood-gate-wait phase context", err)
	}
}

func TestRecordDelayGatesNextCall(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = io.WriteString(w, "<ok/>")
	}))
	defer srv.Close()

	fc := newFakeClock()
	c := newClient(srv, fc)
	c.RecordDelay(500 * time.Millisecond)

	if _, err := c.Do(context.Background(), srv.URL, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
	naps := fc.Naps()
	if len(naps) != 1 || naps[0] != 500*time.Millisecond {
		t.Errorf("naps = %v, want [500ms]", naps)
	}

	// A second Do without recording a fresh delay must not sleep again.
	if _, err := c.Do(context.Background(), srv.URL, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if got := len(fc.Naps()); got != 1 {
		t.Errorf("naps after second call = %d, want 1", got)
	}
}

func TestRecordDelayZeroClears(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "<ok/>")
	}))
	defer srv.Close()

	fc := newFakeClock()
	c := newClient(srv, fc)
	c.RecordDelay(time.Second)
	c.RecordDelay(0)

	if _, err := c.Do(context.Background(), srv.URL, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if naps := fc.Naps(); len(naps) != 0 {
		t.Errorf("naps = %v, want none after RecordDelay(0)", naps)
	}
}

func TestRecordDelayKeepsLongerExistingDelay(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "<ok/>")
	}))
	defer srv.Close()

	fc := newFakeClock()
	c := newClient(srv, fc)
	// A long gate followed by a shorter one (the fakeClock does not
	// advance between RecordDelay calls): the shorter delay must not
	// shorten the still-pending longer gate.
	c.RecordDelay(time.Second)
	c.RecordDelay(100 * time.Millisecond)

	if _, err := c.Do(context.Background(), srv.URL, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
	naps := fc.Naps()
	if len(naps) != 1 || naps[0] != time.Second {
		t.Errorf("naps = %v, want [1s] (longer gate preserved, not reduced to 100ms)", naps)
	}
}

func TestRecordDelayExtendsWhenNewDelayIsLonger(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "<ok/>")
	}))
	defer srv.Close()

	fc := newFakeClock()
	c := newClient(srv, fc)
	// A short gate followed by a longer one: the gate must extend to the
	// later deadline.
	c.RecordDelay(100 * time.Millisecond)
	c.RecordDelay(time.Second)

	if _, err := c.Do(context.Background(), srv.URL, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
	naps := fc.Naps()
	if len(naps) != 1 || naps[0] != time.Second {
		t.Errorf("naps = %v, want [1s] (gate extended to the longer delay)", naps)
	}
}

func TestDoRespectsContextDeadlineDuringRequest(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		<-r.Context().Done()
	}))
	defer srv.Close()

	c := newClient(srv, newFakeClock())
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := c.Do(ctx, srv.URL, nil)
	if err == nil {
		t.Fatal("expected error on cancelled context")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want context.DeadlineExceeded preserved", err)
	}
	// The caller's cancellation is not a transient server condition:
	// even with MaxRetries at its default, exactly one attempt runs.
	if hits.Load() != 1 {
		t.Errorf("requests = %d, want 1 (no retry on context cancellation)", hits.Load())
	}
}

// The 16-MB soap.MaxResponseBytes cap must bite at the transport read:
// the api/auth clients buffer the whole body here before the capped
// soap decoders ever run, so this is the only place the guard can
// actually prevent memory exhaustion. Oversize is not retryable.
func TestDoRejectsOversizedResponse(t *testing.T) {
	var hits atomic.Int32
	chunk := bytes.Repeat([]byte("x"), 1<<20)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		for i := 0; i < 17; i++ {
			if _, err := w.Write(chunk); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	c := newClient(srv, newFakeClock())
	_, err := c.Do(context.Background(), srv.URL, []byte(sampleEnvelope))
	if err == nil {
		t.Fatal("Do: want oversize error, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Errorf("err = %v, want a response-size error", err)
	}
	if hits.Load() != 1 {
		t.Errorf("requests = %d, want 1 (oversize must not be retried)", hits.Load())
	}
}

// A per-attempt HTTPClient.Timeout error matches
// errors.Is(err, context.DeadlineExceeded) even though the caller's ctx
// is still alive; it is a transient slow-server condition and must stay
// retryable — only the caller's own cancellation suppresses the retry.
func TestDoRetriesOnClientTimeout(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if hits.Add(1) == 1 {
			// Outlast the 100ms client timeout, then return so
			// srv.Close does not wait on a stuck handler.
			time.Sleep(500 * time.Millisecond)
			return
		}
		_, _ = io.WriteString(w, "<ok/>")
	}))
	defer srv.Close()

	c := newClient(srv, newFakeClock())
	c.HTTPClient.Timeout = 100 * time.Millisecond
	resp, err := c.Do(context.Background(), srv.URL, []byte(sampleEnvelope))
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if string(resp) != "<ok/>" {
		t.Errorf("body = %q, want <ok/>", resp)
	}
	if hits.Load() != 2 {
		t.Errorf("requests = %d, want 2 (client-timeout attempt retried once)", hits.Load())
	}
}
