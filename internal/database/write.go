package database

import (
	"context"
	"errors"

	"github.com/chmmou/kasapi-cli/internal/kaswrite"
)

// ErrUnexpectedReturnString is the shared canonical post-call-contract
// sentinel, re-exported so errors.Is(err, database.ErrUnexpectedReturnString)
// keeps working and the slice stays self-describing. See
// kaswrite.ErrUnexpectedReturnString for the full contract.
var ErrUnexpectedReturnString = kaswrite.ErrUnexpectedReturnString

const (
	addAction    = "add_database"
	updateAction = "update_database"
	deleteAction = "delete_database"
)

// The Field-prefixed constants are the KAS request keys add_database /
// update_database accept besides the database_login identifier;
// AddParams reuses the subset add expects. Each update field is an
// optional wholesale replacement: only the keys the caller explicitly
// sets are sent, so an empty string is a meaningful value rather than
// "leave unchanged".
//
// The password key differs between actions: add_database sets the
// initial password via database_password, update_database replaces it
// via database_new_password. Both keys are kept explicit so neither
// path can silently send the wrong one (the same _new_password split
// the ftpuser/sambauser slices carry).
const (
	FieldLogin        = "database_login"
	FieldPassword     = "database_password"
	FieldNewPassword  = "database_new_password"
	FieldComment      = "database_comment"
	FieldAllowedHosts = "database_allowed_hosts"
)

// Spec carries the add_database request fields. Password, comment and
// allowed_hosts are required by the KAS API and validated as non-empty
// before any SOAP call so the CLI can surface a fast validation error.
// The allowed_hosts grammar (comma-separated host names / IPs / CIDR)
// is delegated to the API — a wrong value surfaces as
// database_allowed_hosts_syntax_incorrect.
//
// add_database takes no database_login: the server auto-generates the
// login (always equal to the database name on creation, e.g.
// "d0123460") and echoes it in ReturnInfo, which Add returns.
type Spec struct {
	Password     string
	Comment      string
	AllowedHosts string
}

// Add creates a database (add_database) and returns the login the
// server assigns, as echoed in ReturnInfo (e.g. "d0123460").
//
// Password, comment and allowed_hosts are validated before the SOAP
// call so an obviously incomplete spec fails fast; the remaining syntax
// validation (password_syntax_incorrect,
// database_allowed_hosts_syntax_incorrect, …) is left to the API and
// surfaces verbatim through the Caller.
func (cl *Client) Add(ctx context.Context, s Spec) (string, error) {
	if s.Password == "" || s.Comment == "" || s.AllowedHosts == "" {
		return "", errors.New("database: add_database requires a non-empty password, comment and allowed_hosts")
	}
	resp, err := kaswrite.Call(ctx, cl.c, "database", addAction, AddParams(s))
	if err != nil {
		return "", err
	}
	return resp.Body.ReturnInfo.AsString(), nil
}

// AddParams builds the add_database KAS request parameter map. It is
// the single source of truth for the request shape so the CLI dry-run
// preview / audit record and the dispatched call cannot diverge. No
// database_login is sent — the server generates it. The password is
// sent under database_password; update uses database_new_password
// instead.
func AddParams(s Spec) map[string]any {
	return map[string]any{
		FieldPassword:     s.Password,
		FieldComment:      s.Comment,
		FieldAllowedHosts: s.AllowedHosts,
	}
}

// Update changes one or more mutable fields of an existing database
// (update_database). fields holds only the keys the caller wants to
// change (use the Field* constants; the password is FieldNewPassword
// here, not FieldPassword); each is applied wholesale. login and at
// least one field are required — update_database with nothing to
// change is rejected before the SOAP call (the API would fault
// nothing_to_do).
func (cl *Client) Update(ctx context.Context, login string, fields map[string]string) error {
	if login == "" {
		return errors.New("database: update_database requires a non-empty database login")
	}
	if len(fields) == 0 {
		return errors.New("database: update_database requires at least one field to change")
	}
	_, err := kaswrite.Call(ctx, cl.c, "database", updateAction, UpdateParams(login, fields))
	return err
}

// UpdateParams builds the update_database KAS request parameter map
// (single source of truth, see AddParams): the database_login
// identifier plus every caller-supplied mutable field verbatim.
func UpdateParams(login string, fields map[string]string) map[string]any {
	params := map[string]any{FieldLogin: login}
	for k, v := range fields {
		params[k] = v
	}
	return params
}

// Delete removes a database (delete_database). The action is highly
// destructive — it drops the database and every row in it. The CLI
// gates it behind the #109 confirmation prompt; a SOAP fault (e.g.
// database_login_not_found, in_progress) is surfaced verbatim by the
// Caller so the caller can classify it via the api error helpers.
func (cl *Client) Delete(ctx context.Context, login string) error {
	if login == "" {
		return errors.New("database: delete_database requires a non-empty database login")
	}
	_, err := kaswrite.Call(ctx, cl.c, "database", deleteAction, DeleteParams(login))
	return err
}

// DeleteParams builds the delete_database KAS request parameter map
// (single source of truth, see AddParams).
func DeleteParams(login string) map[string]any {
	return map[string]any{FieldLogin: login}
}
