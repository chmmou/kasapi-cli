package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/transport"
)

// DefaultEndpoint is the production KasApi.php URL. Set Client.Endpoint
// to override (e.g. in tests against an httptest.Server).
const DefaultEndpoint = "https://kasapi.kasserver.com/soap/KasApi.php"

// floodFallback is the gate applied when the server returns a
// flood_protection fault, which carries no KasFloodDelay value.
const floodFallback = 2 * time.Second

// TokenSource supplies the credentials for one or more KasApi calls.
// Plain auth returns the password as AuthData; session auth returns a
// short-lived 40-char token. After an authentication failure, Client
// calls Invalidate so the next Credentials call may obtain a fresh
// token (e.g. by re-running the KasAuth flow).
type TokenSource interface {
	Credentials(ctx context.Context) (login, authData string, authType soap.AuthType, err error)
	Invalidate()
}

// Heartbeater is an optional TokenSource extension. After every
// successful Call, Client invokes Heartbeat so a session-token source
// configured with session_update_lifetime=Y can extend its locally
// cached expiry to mirror the rolling window the server applies.
type Heartbeater interface {
	Heartbeat()
}

// StaticTokenSource is a TokenSource that returns the same credentials
// every call. Suitable for plain auth and for tests; session-token
// callers should use the source provided by the auth package once the
// KasAuth client (issue #5) is in place.
type StaticTokenSource struct {
	Login    string
	AuthData string
	AuthType soap.AuthType
}

// Credentials returns the stored values without modification.
func (s *StaticTokenSource) Credentials(_ context.Context) (string, string, soap.AuthType, error) {
	if s.Login == "" || s.AuthData == "" || s.AuthType == "" {
		return "", "", "", errors.New("api: StaticTokenSource has empty fields")
	}
	return s.Login, s.AuthData, s.AuthType, nil
}

// Invalidate is a no-op for the static source. A static credential
// cannot refresh itself; an auth failure is therefore terminal.
func (s *StaticTokenSource) Invalidate() {}

// Client posts KasApi calls through the transport, refreshes session
// tokens on auth failures, and feeds the server-reported KasFloodDelay
// back to the transport gate after every successful call.
//
// A zero Client is unusable; obtain one with New.
type Client struct {
	Transport *transport.Client
	Tokens    TokenSource
	Endpoint  string
}

// New returns a Client wired with the given transport and token source.
// Endpoint defaults to DefaultEndpoint.
func New(t *transport.Client, ts TokenSource) *Client {
	return &Client{
		Transport: t,
		Tokens:    ts,
		Endpoint:  DefaultEndpoint,
	}
}

// Call posts one KasApi action with the given parameters and returns
// the parsed response. SOAP-ENV:Fault bodies surface as *Error with the
// stable KAS code in Error.Code. On no_auth or unknown_session, Call
// invalidates the token source and retries once with fresh credentials.
//
// On flood_protection, Call seeds the transport gate with a short
// fallback delay before returning the error so the caller's next Call
// is throttled even though the fault carried no KasFloodDelay value.
func (c *Client) Call(ctx context.Context, action string, params map[string]any) (*soap.Response, error) {
	if action == "" {
		return nil, errors.New("api: action is required")
	}
	resp, err := c.callOnce(ctx, action, params)
	if err != nil && IsAuthFailure(err) {
		c.Tokens.Invalidate()
		resp, err = c.callOnce(ctx, action, params)
	}
	if err == nil {
		if hb, ok := c.Tokens.(Heartbeater); ok {
			hb.Heartbeat()
		}
		return resp, nil
	}
	return nil, err
}

func (c *Client) callOnce(ctx context.Context, action string, params map[string]any) (*soap.Response, error) {
	login, authData, authType, err := c.Tokens.Credentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("api: credentials: %w", err)
	}

	var buf bytes.Buffer
	req := soap.Request{
		Login:    login,
		AuthType: authType,
		AuthData: authData,
		Action:   action,
		Params:   params,
	}
	if encErr := soap.EncodeRequest(&buf, req); encErr != nil {
		return nil, fmt.Errorf("api: encode %s: %w", action, encErr)
	}

	body, err := c.Transport.Do(ctx, c.Endpoint, buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("api: transport %s: %w", action, err)
	}

	resp, err := soap.Decode(bytes.NewReader(body))
	if err != nil {
		var fe *soap.FaultError
		if errors.As(err, &fe) {
			apiErr := newError(action, c.Endpoint, fe)
			if IsFloodProtection(apiErr) {
				c.Transport.RecordDelay(floodFallback)
			}
			return nil, apiErr
		}
		return nil, fmt.Errorf("api: decode %s: %w", action, err)
	}

	if d := time.Duration(resp.Body.KasFloodDelay * float64(time.Second)); d > 0 {
		c.Transport.RecordDelay(d)
	}
	return resp, nil
}
