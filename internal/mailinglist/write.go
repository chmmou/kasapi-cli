package mailinglist

import (
	"context"
	"errors"
	"fmt"

	"github.com/chmmou/kasapi-cli/internal/soap"
)

// ErrUnexpectedReturnString is returned by the write methods when the
// KAS call succeeds at the transport level (no SOAP fault) but the
// server's ReturnString is not "TRUE" — an API-drift / contract
// violation. Callers may errors.Is against it to tell a protocol-shape
// regression apart from a transport or KAS-fault error; the wrapped
// message carries the action and the observed value. Mirrors
// mailforward.ErrUnexpectedReturnString.
var ErrUnexpectedReturnString = errors.New("mailinglist: unexpected ReturnString (want TRUE)")

const (
	addAction    = "add_mailinglist"
	updateAction = "update_mailinglist"
	deleteAction = "delete_mailinglist"
)

// The Field-prefixed constants are the KAS request keys
// update_mailinglist accepts besides the mailinglist_name identifier.
// FieldSubscriber / FieldRestrictPost are RFC2822 address lists
// (newline-separated), FieldConfig is the complete list configuration
// as plain text, and FieldIsActive toggles the list (Y|N). Each is an
// optional wholesale replacement; only the keys the caller explicitly
// sets are sent, so an empty string is a meaningful "clear" rather than
// "leave unchanged".
const (
	FieldSubscriber   = "subscriber"
	FieldRestrictPost = "restrict_post"
	FieldConfig       = "config"
	FieldIsActive     = "is_active"
)

// Add creates a mailing list named name under domain (add_mailinglist)
// with the given list password. It returns the canonical list
// identifier the server assigns, as echoed in ReturnInfo (e.g.
// "announce-example-org").
//
// name, domain and password are all required; an empty value is
// rejected before any SOAP call so the CLI can surface a fast
// validation error.
func (cl *Client) Add(ctx context.Context, name, domain, password string) (string, error) {
	if name == "" || domain == "" {
		return "", errors.New("mailinglist: add_mailinglist requires a non-empty name and domain")
	}
	if password == "" {
		return "", errors.New("mailinglist: add_mailinglist requires a non-empty password")
	}
	resp, err := cl.call(ctx, addAction, AddParams(name, domain, password))
	if err != nil {
		return "", err
	}
	return resp.Body.ReturnInfo.AsString(), nil
}

// AddParams builds the add_mailinglist KAS request parameter map. It is
// the single source of truth for the request shape so the CLI dry-run
// preview / audit record and the dispatched call cannot diverge.
func AddParams(name, domain, password string) map[string]any {
	return map[string]any{
		"mailinglist_name":     name,
		"mailinglist_domain":   domain,
		"mailinglist_password": password,
	}
}

// Update changes one or more mutable fields of an existing mailing list
// (update_mailinglist). fields holds only the keys the caller wants to
// change (use the Field* constants); each is applied wholesale. name
// and at least one field are required — update_mailinglist with nothing
// to change is rejected before the SOAP call (the API would fault
// nothing_to_do).
func (cl *Client) Update(ctx context.Context, name string, fields map[string]string) error {
	if name == "" {
		return errors.New("mailinglist: update_mailinglist requires a non-empty mailing list name")
	}
	if len(fields) == 0 {
		return errors.New("mailinglist: update_mailinglist requires at least one field to change")
	}
	_, err := cl.call(ctx, updateAction, UpdateParams(name, fields))
	return err
}

// UpdateParams builds the update_mailinglist KAS request parameter map
// (single source of truth, see AddParams): the mailinglist_name
// identifier plus every caller-supplied mutable field verbatim.
func UpdateParams(name string, fields map[string]string) map[string]any {
	params := map[string]any{"mailinglist_name": name}
	for k, v := range fields {
		params[k] = v
	}
	return params
}

// Delete removes a mailing list (delete_mailinglist). A SOAP fault
// (e.g. mailinglist_not_found, in_progress) is surfaced verbatim by the
// Caller so the caller can classify it via the api error helpers.
func (cl *Client) Delete(ctx context.Context, name string) error {
	if name == "" {
		return errors.New("mailinglist: delete_mailinglist requires a non-empty mailing list name")
	}
	_, err := cl.call(ctx, deleteAction, DeleteParams(name))
	return err
}

// DeleteParams builds the delete_mailinglist KAS request parameter map
// (single source of truth, see AddParams).
func DeleteParams(name string) map[string]any {
	return map[string]any{"mailinglist_name": name}
}

// call dispatches a write action and enforces the shared post-call
// contract: a non-fault response must echo ReturnString="TRUE";
// anything else wraps ErrUnexpectedReturnString so a future API drift
// fails the mapping test instead of silently passing.
func (cl *Client) call(ctx context.Context, action string, params map[string]any) (*soap.Response, error) {
	resp, err := cl.c.Call(ctx, action, params)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("mailinglist: %s: nil response without error from Caller", action)
	}
	if got := resp.Body.ReturnString; got != "TRUE" {
		return nil, fmt.Errorf("%w: %s got %q", ErrUnexpectedReturnString, action, got)
	}
	return resp, nil
}
