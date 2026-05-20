package ddns

import (
	"context"
	"errors"

	"github.com/chmmou/kasapi-cli/internal/kaswrite"
)

// ErrUnexpectedReturnString is the shared canonical post-call-contract
// sentinel, re-exported so errors.Is(err, ddns.ErrUnexpectedReturnString)
// keeps working and the slice stays self-describing. See
// kaswrite.ErrUnexpectedReturnString for the full contract.
var ErrUnexpectedReturnString = kaswrite.ErrUnexpectedReturnString

const (
	addAction    = "add_ddnsuser"
	updateAction = "update_ddnsuser"
	deleteAction = "delete_ddnsuser"
)

// The Field-prefixed constants are the KAS request keys add_ddnsuser /
// update_ddnsuser accept besides the dyndns_login identifier; AddParams
// reuses the subset add expects. Each update field is an optional
// wholesale replacement: only the keys the caller explicitly sets are
// sent, so an empty string is a meaningful value rather than
// "leave unchanged".
//
// Wire-side asymmetry mirrors the read path: the action name uses
// "ddnsuser" / "ddnsusers" (no y), the get filter uses "ddns_login"
// (no y), but every other request key on the write actions uses the
// "dyndns_*" prefix (with y). The Field* constants follow the wire,
// not the action.
//
// Unlike ftpuser/sambauser there is no _new_password asymmetry — both
// add_ddnsuser and update_ddnsuser carry the password under
// FieldPassword (dyndns_password). A single constant is therefore
// sufficient.
//
// NOTE: FieldTargetIPv4 and FieldTargetIPv6 are *undocumented* for
// update_ddnsuser but verified to work against the live API (observed
// in the KAS panel's browser network tab). The captured
// update_ddnsuser_request.xml (plus its success-response request echo)
// is the authoritative request-shape contract for this slice — when
// the public docs catch up they should match the fixture, not the
// other way round. FieldTargetIP is the legacy single-IP key used by
// add_ddnsuser; update uses the dual-stack ipv4/ipv6 pair instead.
const (
	FieldLogin      = "dyndns_login"
	FieldPassword   = "dyndns_password"
	FieldZone       = "dyndns_zone"
	FieldLabel      = "dyndns_label"
	FieldTargetIP   = "dyndns_target_ip"
	FieldTargetIPv4 = "dyndns_target_ipv4"
	FieldTargetIPv6 = "dyndns_target_ipv6"
	FieldDualStack  = "dyndns_dual_stack"
	FieldComment    = "dyndns_comment"
)

// Spec carries the add_ddnsuser request fields. Password, zone, label,
// target IP and comment are required by the KAS API and validated as
// non-empty before any SOAP call so the CLI can surface a fast
// validation error. dual_stack defaults to "N" (single-stack) on the
// CLI side and is passed through verbatim — the API only accepts the
// literal strings "Y" / "N".
//
// add_ddnsuser takes no dyndns_login: the server auto-generates the
// login and echoes it in ReturnInfo (e.g. "dyn0000001"), which Add
// returns.
type Spec struct {
	Password  string
	Zone      string
	Label     string
	TargetIP  string
	Comment   string
	DualStack string
}

// Add creates a DDNS user (add_ddnsuser) and returns the login the
// server assigns, as echoed in ReturnInfo (e.g. "dyn0000001").
//
// Password, zone, label, target IP and comment are validated before the
// SOAP call so an obviously incomplete spec fails fast; the remaining
// syntax validation (record_name_syntax_incorrect,
// dyndns_target_ip_syntax_incorrect, …) is left to the API and surfaces
// verbatim through the Caller.
func (cl *Client) Add(ctx context.Context, s Spec) (string, error) {
	if s.Password == "" || s.Zone == "" || s.Label == "" || s.TargetIP == "" || s.Comment == "" {
		return "", errors.New("ddns: add_ddnsuser requires a non-empty password, zone, label, target IP and comment")
	}
	resp, err := kaswrite.Call(ctx, cl.c, "ddns", addAction, AddParams(s))
	if err != nil {
		return "", err
	}
	return resp.Body.ReturnInfo.AsString(), nil
}

// AddParams builds the add_ddnsuser KAS request parameter map. It is
// the single source of truth for the request shape so the CLI dry-run
// preview / audit record and the dispatched call cannot diverge. No
// dyndns_login is sent — the server generates it. add_ddnsuser uses
// the legacy single-IP FieldTargetIP key, not the ipv4/ipv6 pair.
func AddParams(s Spec) map[string]any {
	return map[string]any{
		FieldPassword:  s.Password,
		FieldZone:      s.Zone,
		FieldLabel:     s.Label,
		FieldTargetIP:  s.TargetIP,
		FieldComment:   s.Comment,
		FieldDualStack: s.DualStack,
	}
}

// Update changes one or more mutable fields of an existing DDNS user
// (update_ddnsuser). fields holds only the keys the caller wants to
// change (use the Field* constants); each is applied wholesale. login
// and at least one field are required — update_ddnsuser with nothing to
// change is rejected before the SOAP call (the API would fault
// nothing_to_do).
func (cl *Client) Update(ctx context.Context, login string, fields map[string]string) error {
	if login == "" {
		return errors.New("ddns: update_ddnsuser requires a non-empty dyndns login")
	}
	if len(fields) == 0 {
		return errors.New("ddns: update_ddnsuser requires at least one field to change")
	}
	_, err := kaswrite.Call(ctx, cl.c, "ddns", updateAction, UpdateParams(login, fields))
	return err
}

// UpdateParams builds the update_ddnsuser KAS request parameter map
// (single source of truth, see AddParams): the dyndns_login identifier
// plus every caller-supplied mutable field verbatim.
func UpdateParams(login string, fields map[string]string) map[string]any {
	params := map[string]any{FieldLogin: login}
	for k, v := range fields {
		params[k] = v
	}
	return params
}

// Delete removes a DDNS user (delete_ddnsuser). A SOAP fault (e.g.
// dyndns_login_not_found) is surfaced verbatim by the Caller so the
// caller can classify it via the api error helpers.
func (cl *Client) Delete(ctx context.Context, login string) error {
	if login == "" {
		return errors.New("ddns: delete_ddnsuser requires a non-empty dyndns login")
	}
	_, err := kaswrite.Call(ctx, cl.c, "ddns", deleteAction, DeleteParams(login))
	return err
}

// DeleteParams builds the delete_ddnsuser KAS request parameter map
// (single source of truth, see AddParams).
func DeleteParams(login string) map[string]any {
	return map[string]any{FieldLogin: login}
}
