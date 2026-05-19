package sambauser

import (
	"context"
	"errors"

	"github.com/chmmou/kasapi-cli/internal/kaswrite"
)

// ErrUnexpectedReturnString is the shared canonical post-call-contract
// sentinel, re-exported so errors.Is(err, sambauser.ErrUnexpectedReturnString)
// keeps working and the slice stays self-describing. See
// kaswrite.ErrUnexpectedReturnString for the full contract.
var ErrUnexpectedReturnString = kaswrite.ErrUnexpectedReturnString

const (
	addAction    = "add_sambauser"
	updateAction = "update_sambauser"
	deleteAction = "delete_sambauser"
)

// The Field-prefixed constants are the KAS request keys update_sambauser
// accepts besides the samba_login identifier; AddParams reuses the
// shared subset. Each is an optional wholesale replacement on update:
// only the keys the caller explicitly sets are sent, so an empty string
// is a meaningful value rather than "leave unchanged".
//
// The password key differs between actions: add_sambauser sets the
// initial password via samba_password, update_sambauser replaces it via
// samba_new_password. Both keys are kept explicit so neither path can
// silently send the wrong one. NOTE: the KAS documentation's
// add_sambauser parameter table wrongly lists samba_new_password for
// the create call; the captured add_sambauser request fixture (and its
// success-response request echo) confirm the real key is samba_password
// — the fixture is the authoritative request-shape contract here.
const (
	FieldLogin       = "samba_login"
	FieldPassword    = "samba_password"
	FieldNewPassword = "samba_new_password"
	FieldPath        = "samba_path"
	FieldComment     = "samba_comment"
)

// Spec carries the add_sambauser request fields. All three are required
// by the KAS API and validated as non-empty before any SOAP call so the
// CLI can surface a fast validation error.
//
// add_sambauser takes no samba_login: the server auto-generates the
// login and echoes it in ReturnInfo (e.g. "s0000003"), which Add
// returns.
type Spec struct {
	Password string
	Comment  string
	Path     string
}

// Add creates a Samba user (add_sambauser) and returns the login the
// server assigns, as echoed in ReturnInfo (e.g. "s0000003").
//
// Password, comment and path are validated before the SOAP call so an
// obviously incomplete spec fails fast; the remaining syntax validation
// (path_syntax_incorrect, password_syntax_incorrect, …) is left to the
// API and surfaces verbatim through the Caller.
func (cl *Client) Add(ctx context.Context, s Spec) (string, error) {
	if s.Password == "" || s.Comment == "" || s.Path == "" {
		return "", errors.New("sambauser: add_sambauser requires a non-empty password, comment and path")
	}
	resp, err := kaswrite.Call(ctx, cl.c, "sambauser", addAction, AddParams(s))
	if err != nil {
		return "", err
	}
	return resp.Body.ReturnInfo.AsString(), nil
}

// AddParams builds the add_sambauser KAS request parameter map. It is
// the single source of truth for the request shape so the CLI dry-run
// preview / audit record and the dispatched call cannot diverge. No
// samba_login is sent — the server generates it. The password is sent
// under samba_password (see the Field* doc comment for the
// documentation discrepancy).
func AddParams(s Spec) map[string]any {
	return map[string]any{
		FieldPassword: s.Password,
		FieldComment:  s.Comment,
		FieldPath:     s.Path,
	}
}

// Update changes one or more mutable fields of an existing Samba user
// (update_sambauser). fields holds only the keys the caller wants to
// change (use the Field* constants; the password is FieldNewPassword
// here, not FieldPassword); each is applied wholesale. login and at
// least one field are required — update_sambauser with nothing to
// change is rejected before the SOAP call (the API would fault
// nothing_to_do).
func (cl *Client) Update(ctx context.Context, login string, fields map[string]string) error {
	if login == "" {
		return errors.New("sambauser: update_sambauser requires a non-empty samba login")
	}
	if len(fields) == 0 {
		return errors.New("sambauser: update_sambauser requires at least one field to change")
	}
	_, err := kaswrite.Call(ctx, cl.c, "sambauser", updateAction, UpdateParams(login, fields))
	return err
}

// UpdateParams builds the update_sambauser KAS request parameter map
// (single source of truth, see AddParams): the samba_login identifier
// plus every caller-supplied mutable field verbatim.
func UpdateParams(login string, fields map[string]string) map[string]any {
	params := map[string]any{FieldLogin: login}
	for k, v := range fields {
		params[k] = v
	}
	return params
}

// Delete removes a Samba user (delete_sambauser). A SOAP fault (e.g.
// samba_login_not_found, in_progress) is surfaced verbatim by the
// Caller so the caller can classify it via the api error helpers.
func (cl *Client) Delete(ctx context.Context, login string) error {
	if login == "" {
		return errors.New("sambauser: delete_sambauser requires a non-empty samba login")
	}
	_, err := kaswrite.Call(ctx, cl.c, "sambauser", deleteAction, DeleteParams(login))
	return err
}

// DeleteParams builds the delete_sambauser KAS request parameter map
// (single source of truth, see AddParams).
func DeleteParams(login string) map[string]any {
	return map[string]any{FieldLogin: login}
}
