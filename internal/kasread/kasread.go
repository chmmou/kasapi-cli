// Package kasread provides shared scaffolding for the read endpoints
// of the KAS API. The scaffolding captures the canonical "fetch a
// list, or a singular variant filtered by one field" shape that the
// majority of internal/<module>.Client types need; modules consume
// it from their own NewClient and expose List/Get methods that
// delegate one-line.
package kasread

import (
	"context"
	"fmt"

	"github.com/chmmou/kasapi-cli/internal/soap"
)

// Caller is the subset of *api.Client that the read scaffolding
// depends on. Each domain module previously redefined this same
// interface; collecting it here removes the duplication while
// keeping the dependency-inversion property — modules and tests
// pass any value with a matching Call method (e.g. the shared
// testutil.FakeCaller).
type Caller interface {
	Call(ctx context.Context, action string, params map[string]any) (*soap.Response, error)
}

// ListGet binds a KAS read action to its singular-variant filter
// parameter and a typed decoder. The List/Get methods then drive
// the canonical fetch + decode + first-or-not-found pipeline that
// nearly every module's Client repeated by hand.
//
// L is the named list type (e.g. ddns.DDNSUserList); E is the
// element type (e.g. ddns.DDNSUser). Decoder is the module's
// existing Decode<Foo>s function — its signature already matches
// the field, so callers pass it directly.
type ListGet[L ~[]E, E any] struct {
	Caller    Caller
	Action    string // KAS action, e.g. "get_ddnsusers".
	Label     string // module-name prefix used in error messages, e.g. "ddns".
	ArgName   string // input name in the empty-input error, e.g. "login".
	FilterKey string // KAS wire parameter name for Get, e.g. "ddns_login".
	Decoder   func(soap.Value) (L, error)
}

// List calls Action with no parameters and returns the decoded
// list. Decode errors are wrapped as "<Label>: <Action>: <err>".
// Transport errors propagate verbatim so callers can keep using
// api.IsNotFound and the other transport-error helpers.
func (lg ListGet[L, E]) List(ctx context.Context) (L, error) {
	resp, err := lg.Caller.Call(ctx, lg.Action, nil)
	if err != nil {
		return nil, err
	}
	list, err := lg.Decoder(resp.Body.ReturnInfo)
	if err != nil {
		return nil, fmt.Errorf("%s: %s: %w", lg.Label, lg.Action, err)
	}
	return list, nil
}

// Get calls Action with {FilterKey: value} and returns the single
// matching entry. An empty response yields a "<Label>: %q not found"
// error; an ambiguous response with more than one entry yields a
// "<Label>: %q matched N entries (expected unique)" error rather than
// silently returning the first. An empty value yields "<Label>:
// <ArgName> is required" without performing the call.
func (lg ListGet[L, E]) Get(ctx context.Context, value string) (E, error) {
	var zero E
	if value == "" {
		return zero, fmt.Errorf("%s: %s is required", lg.Label, lg.ArgName)
	}
	resp, err := lg.Caller.Call(ctx, lg.Action, map[string]any{lg.FilterKey: value})
	if err != nil {
		return zero, err
	}
	list, err := lg.Decoder(resp.Body.ReturnInfo)
	if err != nil {
		return zero, fmt.Errorf("%s: %s: %w", lg.Label, lg.Action, err)
	}
	if len(list) == 0 {
		return zero, fmt.Errorf("%s: %q not found", lg.Label, value)
	}
	if len(list) > 1 {
		return zero, fmt.Errorf("%s: %q matched %d entries (expected unique)", lg.Label, value, len(list))
	}
	return list[0], nil
}
