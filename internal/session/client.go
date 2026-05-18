package session

import (
	"context"

	"github.com/chmmou/kasapi-cli/internal/kaswrite"
)

// ErrUnexpectedReturnString is the shared canonical post-call-contract
// sentinel, re-exported so errors.Is(err, session.ErrUnexpectedReturnString)
// keeps working and the slice stays self-describing. See
// kaswrite.ErrUnexpectedReturnString for the full contract.
var ErrUnexpectedReturnString = kaswrite.ErrUnexpectedReturnString

// Caller is the subset of *api.Client this package's KAS-side use case
// depends on. delete_session dispatches through the kaswrite write
// seam, so the alias resolves there (kaswrite.Caller is the same
// underlying type as kasread.Caller); reusing it keeps tests free of
// network setup: a testutil.FakeCaller can return a *soap.Response
// decoded from a fixture.
type Caller = kaswrite.Caller

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
// shared kaswrite seam enforces the post-call contract: the server
// must echo ReturnString="TRUE"; any other value wraps
// ErrUnexpectedReturnString so a future API drift fails the mapping
// test instead of silently passing.
func (cl *Client) Delete(ctx context.Context) error {
	_, err := kaswrite.Call(ctx, cl.c, "session", deleteSessionAction, nil)
	return err
}
