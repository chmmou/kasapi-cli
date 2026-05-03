package transport

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/chmmou/kasapi-cli/internal/version"
)

// Defaults applied by New.
const (
	DefaultTimeout    = 30 * time.Second
	DefaultMaxRetries = 3
	defaultBackoff    = 500 * time.Millisecond
)

// DefaultUserAgent identifies the CLI to the KAS server. The version
// suffix is filled at build time via -ldflags.
var DefaultUserAgent = "kasapi-cli/" + version.Version + " (+https://github.com/chmmou/kasapi-cli)"

// Client posts SOAP envelopes over HTTPS and gates outgoing requests
// behind a configurable per-instance delay. Callers report the next
// delay via RecordDelay after parsing the response body.
//
// A zero Client is unusable; obtain one with New.
type Client struct {
	HTTPClient *http.Client
	UserAgent  string
	MaxRetries int

	// Now and Sleep are overridable for tests so retry/flood-delay
	// timing can be asserted without real wall-clock waits.
	Now   func() time.Time
	Sleep func(ctx context.Context, d time.Duration) error

	mu           sync.Mutex
	nextEarliest time.Time
}

// New returns a Client with sensible defaults: a 30s HTTP timeout, a
// version-stamped User-Agent, and three retries on 5xx / network errors.
func New() *Client {
	return &Client{
		HTTPClient: &http.Client{Timeout: DefaultTimeout},
		UserAgent:  DefaultUserAgent,
		MaxRetries: DefaultMaxRetries,
		Now:        time.Now,
		Sleep:      ctxSleep,
	}
}

// RecordDelay schedules a window during which subsequent calls to Do
// will block. d is the KasFloodDelay reported by the server in the
// most recent response. Negative or zero values clear the gate.
func (c *Client) RecordDelay(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if d <= 0 {
		c.nextEarliest = time.Time{}
		return
	}
	c.nextEarliest = c.now().Add(d)
}

// Do posts body to endpoint, blocking first until any pending flood
// delay has elapsed. On 5xx and network errors, Do retries up to
// MaxRetries times with exponential backoff. 4xx responses and
// context cancellation return immediately.
func (c *Client) Do(ctx context.Context, endpoint string, body []byte) ([]byte, error) {
	if err := c.waitGate(ctx); err != nil {
		return nil, err
	}

	backoff := defaultBackoff
	var lastErr error
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		if attempt > 0 {
			if err := c.Sleep(ctx, backoff); err != nil {
				return nil, err
			}
			backoff *= 2
		}
		resp, err := c.doOnce(ctx, endpoint, body)
		if err == nil {
			return resp, nil
		}
		var rerr *retryableError
		if !errors.As(err, &rerr) {
			return nil, err
		}
		lastErr = rerr.err
	}
	return nil, fmt.Errorf("transport: %d retries exhausted: %w", c.MaxRetries, lastErr)
}

func (c *Client) waitGate(ctx context.Context) error {
	c.mu.Lock()
	var wait time.Duration
	if !c.nextEarliest.IsZero() {
		wait = c.nextEarliest.Sub(c.now())
	}
	c.mu.Unlock()
	if wait <= 0 {
		return nil
	}
	return c.Sleep(ctx, wait)
}

func (c *Client) doOnce(ctx context.Context, endpoint string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("transport: build request: %w", err)
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", "")
	req.Header.Set("User-Agent", c.UserAgent)
	// Intentionally no Accept-Encoding here. net/http sets it to "gzip"
	// itself and transparently decodes the response only when the
	// caller has *not* set the header. Setting it manually disables
	// automatic decompression and hands the gzip stream to the XML
	// decoder, which surfaces as `XML syntax error: invalid character
	// entity` on the first kasserver response that comes back compressed.

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, &retryableError{err: fmt.Errorf("transport: post %s: %w", endpoint, err)}
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &retryableError{err: fmt.Errorf("transport: read body: %w", err)}
	}

	if resp.StatusCode >= 500 {
		return nil, &retryableError{err: fmt.Errorf("transport: %s returned %s", endpoint, resp.Status)}
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("transport: %s returned %s", endpoint, resp.Status)
	}
	return respBody, nil
}

func (c *Client) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

// retryableError marks an error returned by doOnce as eligible for retry.
type retryableError struct{ err error }

func (e *retryableError) Error() string { return e.err.Error() }
func (e *retryableError) Unwrap() error { return e.err }

func ctxSleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
