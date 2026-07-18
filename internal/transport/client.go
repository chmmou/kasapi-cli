package transport

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/chmmou/kasapi-cli/internal/soap"
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

	// Logger receives verbose-mode trace events: gate waits, retries.
	// New() seeds it with a discard logger so callers may write to it
	// unconditionally; cli wires --verbose to a stderr handler.
	Logger *slog.Logger

	// Now and Sleep are overridable for tests so retry/flood-delay
	// timing can be asserted without real wall-clock waits.
	Now   func() time.Time
	Sleep func(ctx context.Context, d time.Duration) error

	mu           sync.Mutex
	nextEarliest time.Time
}

// discardLogger is the shared no-op logger used as the package default
// so callers can invoke Logger / logger() unconditionally. Built once
// at package init.
var discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

// New returns a Client with sensible defaults: a 30s HTTP timeout, a
// version-stamped User-Agent, and three retries on 5xx / network errors.
func New() *Client {
	return &Client{
		HTTPClient: &http.Client{Timeout: DefaultTimeout},
		UserAgent:  DefaultUserAgent,
		MaxRetries: DefaultMaxRetries,
		Logger:     discardLogger,
		Now:        time.Now,
		Sleep:      ctxSleep,
	}
}

// RecordDelay schedules a window during which subsequent calls to Do
// will block. d is the KasFloodDelay reported by the server in the
// most recent response. Negative or zero values clear the gate.
//
// For a positive d the gate is monotonic: it extends to now+d only
// when that is later than an already-pending deadline, so a shorter
// delay arriving while a longer one is still active cannot shorten the
// active gate (which would risk tripping the server's flood
// protection). An explicit zero/negative d still clears unconditionally.
func (c *Client) RecordDelay(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if d <= 0 {
		c.nextEarliest = time.Time{}
		return
	}
	if cand := c.now().Add(d); cand.After(c.nextEarliest) {
		c.nextEarliest = cand
	}
}

// Do posts body to endpoint, blocking first until any pending flood
// delay has elapsed. On 5xx and network errors, Do retries up to
// MaxRetries times with exponential backoff. 4xx responses and
// context cancellation return immediately.
func (c *Client) Do(ctx context.Context, endpoint string, body []byte) ([]byte, error) {
	if err := c.waitGate(ctx); err != nil {
		return nil, err
	}

	// Gate is checked once before the loop, not inside it. 5xx-driven
	// retries do not decode envelopes, so no fresh KasFloodDelay can be
	// recorded between attempts — re-checking the gate would always be a
	// no-op.
	backoff := defaultBackoff
	var lastErr error
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		if attempt > 0 {
			c.logger().Info("transport: retry after transient failure",
				"attempt", attempt, "max", c.MaxRetries, "backoff_ms", backoff.Milliseconds())
			if err := c.Sleep(ctx, backoff); err != nil {
				return nil, fmt.Errorf("transport: retry backoff interrupted: %w", err)
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
	c.logger().Info("transport: waiting for KasFloodDelay gate", "wait_ms", wait.Milliseconds())
	if err := c.Sleep(ctx, wait); err != nil {
		return fmt.Errorf("transport: flood-gate wait interrupted: %w", err)
	}
	return nil
}

func (c *Client) logger() *slog.Logger {
	if c.Logger != nil {
		return c.Logger
	}
	return discardLogger
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
		// The caller's cancelled/expired ctx is a decision to stop, not
		// a transient server condition — retrying would only burn a
		// backoff sleep before failing on the same ctx again. The check
		// is on ctx.Err(), not errors.Is(err, context.DeadlineExceeded):
		// a per-attempt HTTPClient.Timeout error also matches the
		// latter, and that one IS a transient condition worth retrying.
		if ctx.Err() != nil {
			return nil, fmt.Errorf("transport: post %s: %w", endpoint, err)
		}
		return nil, &retryableError{err: fmt.Errorf("transport: post %s: %w", endpoint, err)}
	}
	defer func() { _ = resp.Body.Close() }()

	// The soap decoders cap their input at soap.MaxResponseBytes, but by
	// the time they run the whole body has already been buffered here —
	// so the memory-exhaustion guard must be enforced at this read.
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, soap.MaxResponseBytes+1))
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("transport: read body: %w", err)
		}
		return nil, &retryableError{err: fmt.Errorf("transport: read body: %w", err)}
	}
	if len(respBody) > soap.MaxResponseBytes {
		return nil, fmt.Errorf("transport: response from %s exceeds %d bytes", endpoint, soap.MaxResponseBytes)
	}

	if resp.StatusCode >= 500 {
		// A PHP SOAP server may deliver a SOAP fault with HTTP 500. Such
		// a body must reach the decoder so the typed-fault path (auth
		// refresh, flood fallback, exit-code classification) applies
		// instead of three blind retries that discard the fault. The
		// sniff matches the Fault element with any namespace prefix
		// (SOAP-ENV:Fault, soap:Fault) and the prefix-less
		// default-namespace form. It is a byte-level heuristic: a
		// non-SOAP 5xx body that happens to contain a marker is handed
		// to the decoder and fails there as a non-retryable decode error
		// — accepted, because real gateway/proxy error pages carry
		// neither marker and so stay retryable.
		if bytes.Contains(respBody, []byte(":Fault")) || bytes.Contains(respBody, []byte("<Fault")) {
			return respBody, nil
		}
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
