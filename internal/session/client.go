package session

import (
	"context"
	"fmt"

	"github.com/chmmou/kasapi-cli/internal/kasread"
)

// Caller is the subset of *api.Client this package's KAS-side use case
// depends on. Reusing the shared kasread.Caller keeps tests free of
// network setup: a testutil.FakeCaller can return a *soap.Response
// decoded from a fixture.
type Caller = kasread.Caller

// deleteSessionAction is the KAS action that invalidates the session
// identified by the (login, token) tuple supplied as kas_auth_data /
// kas_auth_type=session. It takes no KasRequestParams.
const deleteSessionAction = "delete_session"

// Client wraps the session-write endpoints of the KAS API that are
// genuinely distinct from the KasAuth credential-token flow. Today that
// is delete_session only; add_session is the KasAuth flow itself and
// lives in internal/auth (see the package doc).
type Client struct {
	c Caller
}

// NewClient returns a Client backed by the given Caller (in production
// an *api.Client wired with a session-token StaticTokenSource).
func NewClient(c Caller) *Client { return &Client{c: c} }

// Delete invalidates the session whose 40-char token the underlying
// Caller authenticates with, by calling kas_action=delete_session with
// no parameters. The KAS API identifies the session via the
// (login, token) tuple in the auth envelope, so nothing is passed in
// KasRequestParams.
//
// A SOAP fault (e.g. unknown_session for an already-dead token) is
// surfaced verbatim by the Caller and returned so the caller can
// classify it via the api error helpers. On a non-fault response the
// server echoes ReturnString="TRUE"; any other value is treated as a
// contract violation so a future API drift fails the mapping test
// instead of silently passing.
func (cl *Client) Delete(ctx context.Context) error {
	resp, err := cl.c.Call(ctx, deleteSessionAction, nil)
	if err != nil {
		return err
	}
	if resp == nil {
		return fmt.Errorf("session: %s: nil response without error from Caller", deleteSessionAction)
	}
	if got := resp.Body.ReturnString; got != "TRUE" {
		return fmt.Errorf("session: %s: unexpected ReturnString %q (want TRUE)", deleteSessionAction, got)
	}
	return nil
}
