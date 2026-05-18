// Package kaswrite provides the shared write-side seam for the KAS API.
// It enforces the canonical post-call contract every write endpoint
// shares — a non-fault response must echo ReturnString="TRUE" — and is
// the write-side counterpart of kasread.ListGet. Domain write packages
// keep their typed Add/Update/Delete validators and dispatch through
// Call, so the nil-guard + ReturnString check lives in exactly one
// place instead of being hand-inlined per module.
package kaswrite

import (
	"context"
	"errors"
	"fmt"

	"github.com/chmmou/kasapi-cli/internal/kasread"
	"github.com/chmmou/kasapi-cli/internal/soap"
)

// Caller is the subset of *api.Client the write seam depends on. It is
// the same interface the read seam already defines, so a single Caller
// type spans both seams (re-export idiom — exactly like session already
// does `type Caller = kasread.Caller`) and tests reuse the shared
// testutil.FakeCaller without redefining the contract.
type Caller = kasread.Caller

// ErrUnexpectedReturnString is returned by Call when the KAS call
// succeeds at the transport level (no SOAP fault) but the server's
// ReturnString is not "TRUE" — an API-drift / contract violation.
// Callers (and the domain write packages that re-export this exact
// value) may errors.Is against it to tell a protocol-shape regression
// apart from a transport or KAS-fault error; the wrapped message
// carries the action and the observed value.
var ErrUnexpectedReturnString = errors.New("kaswrite: unexpected ReturnString (want TRUE)")

// Call dispatches a write action through caller and enforces the shared
// post-call contract: a transport error propagates verbatim; a nil
// response without an error is a Caller-contract violation reported
// with label and action; a non-fault response whose ReturnString is
// not "TRUE" wraps ErrUnexpectedReturnString so a future API drift
// fails the mapping test instead of silently passing. label is the
// domain module name used only for the nil-response diagnostic.
func Call(
	ctx context.Context, caller Caller, label, action string, params map[string]any,
) (*soap.Response, error) {
	resp, err := caller.Call(ctx, action, params)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("%s: %s: nil response without error from Caller", label, action)
	}
	if got := resp.Body.ReturnString; got != "TRUE" {
		return nil, fmt.Errorf("%w: %s got %q", ErrUnexpectedReturnString, action, got)
	}
	return resp, nil
}
