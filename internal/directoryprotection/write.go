package directoryprotection

import (
	"context"
	"errors"

	"github.com/chmmou/kasapi-cli/internal/kaswrite"
)

// ErrUnexpectedReturnString is the shared canonical post-call-contract
// sentinel, re-exported so errors.Is(err,
// directoryprotection.ErrUnexpectedReturnString) keeps working and the
// slice stays self-describing. See kaswrite.ErrUnexpectedReturnString
// for the full contract.
var ErrUnexpectedReturnString = kaswrite.ErrUnexpectedReturnString

const (
	addAction    = "add_directoryprotection"
	updateAction = "update_directoryprotection"
	deleteAction = "delete_directoryprotection"
)

// The Field-prefixed constants are the KAS request keys the write
// actions accept. A directory-protection entry is identified by the
// (directory_path, directory_user) pair; directory_password and
// directory_authname are the mutable fields.
//
// Unlike the database / ftpuser / sambauser slices there is NO
// directory_new_password split: update_directoryprotection takes the
// replacement password under the same directory_password key add uses,
// so a single FieldPassword constant serves both actions (verified
// against the captured update request fixture).
const (
	FieldUser     = "directory_user"
	FieldPath     = "directory_path"
	FieldPassword = "directory_password"
	FieldAuthName = "directory_authname"
)

// Spec carries the add_directoryprotection request fields. User, Path
// and Password are required by the KAS API and validated as non-empty
// before any SOAP call so the CLI can surface a fast validation error;
// AuthName is the optional htaccess realm label (directory_authname)
// and is sent verbatim, empty included.
//
// KAS also accepts directory_user / directory_password as parallel
// arrays to create several protected users in one call (hence the
// directory_user_count_neq_passcount fault). The captured request
// fixtures only exercise the scalar single-user form, and the array
// wire-encoding is not captured, so this slice deliberately models one
// (path, user) protection per call rather than inventing the array
// shape.
type Spec struct {
	User     string
	Path     string
	Password string
	AuthName string
}

// Add creates a directory protection (add_directoryprotection) for one
// (path, user) pair and returns the user as echoed in ReturnInfo.
//
// User, Path and Password are validated before the SOAP call so an
// obviously incomplete spec fails fast; the remaining syntax validation
// (directory_path_syntax_incorrect, directory_password_syntax_incorrect,
// directory_authname_syntax_incorrect, …) is left to the API and
// surfaces verbatim through the Caller.
func (cl *Client) Add(ctx context.Context, s Spec) (string, error) {
	switch {
	case s.User == "":
		return "", errors.New("directoryprotection: add_directoryprotection requires a non-empty directory user")
	case s.Path == "":
		return "", errors.New("directoryprotection: add_directoryprotection requires a non-empty directory path")
	case s.Password == "":
		return "", errors.New("directoryprotection: add_directoryprotection requires a non-empty directory password")
	}
	resp, err := kaswrite.Call(ctx, cl.c, "directoryprotection", addAction, AddParams(s))
	if err != nil {
		return "", err
	}
	return resp.Body.ReturnInfo.AsString(), nil
}

// AddParams builds the add_directoryprotection KAS request parameter
// map. It is the single source of truth for the request shape so the
// CLI dry-run preview / audit record and the dispatched call cannot
// diverge. directory_authname is always sent (an empty realm label is
// a meaningful "no label", not a missing parameter).
func AddParams(s Spec) map[string]any {
	return map[string]any{
		FieldUser:     s.User,
		FieldPath:     s.Path,
		FieldPassword: s.Password,
		FieldAuthName: s.AuthName,
	}
}

// Update changes the mutable fields of an existing directory protection
// (update_directoryprotection). The entry is identified by the
// (path, user) pair; fields holds only the keys the caller wants to
// change (use FieldPassword / FieldAuthName), each applied wholesale.
// At least one field is required — update with nothing to change is
// rejected before the SOAP call (the API would fault nothing_to_do).
func (cl *Client) Update(ctx context.Context, path, user string, fields map[string]string) error {
	switch {
	case path == "":
		return errors.New("directoryprotection: update_directoryprotection requires a non-empty directory path")
	case user == "":
		return errors.New("directoryprotection: update_directoryprotection requires a non-empty directory user")
	}
	if len(fields) == 0 {
		return errors.New("directoryprotection: update_directoryprotection requires at least one field to change")
	}
	_, err := kaswrite.Call(ctx, cl.c, "directoryprotection", updateAction, UpdateParams(path, user, fields))
	return err
}

// UpdateParams builds the update_directoryprotection KAS request
// parameter map (single source of truth, see AddParams): the
// (directory_path, directory_user) identity plus every caller-supplied
// mutable field verbatim.
func UpdateParams(path, user string, fields map[string]string) map[string]any {
	params := map[string]any{
		FieldPath: path,
		FieldUser: user,
	}
	for k, v := range fields {
		params[k] = v
	}
	return params
}

// Delete removes a directory protection (delete_directoryprotection)
// for one (path, user) pair. The action is destructive — it drops the
// access entry. The CLI gates it behind the #109 confirmation prompt; a
// SOAP fault (e.g. in_progress) is surfaced verbatim by the Caller so
// the caller can classify it via the api error helpers.
func (cl *Client) Delete(ctx context.Context, path, user string) error {
	switch {
	case path == "":
		return errors.New("directoryprotection: delete_directoryprotection requires a non-empty directory path")
	case user == "":
		return errors.New("directoryprotection: delete_directoryprotection requires a non-empty directory user")
	}
	_, err := kaswrite.Call(ctx, cl.c, "directoryprotection", deleteAction, DeleteParams(path, user))
	return err
}

// DeleteParams builds the delete_directoryprotection KAS request
// parameter map (single source of truth, see AddParams). Only the
// (directory_path, directory_user) identity is sent — no password or
// authname.
func DeleteParams(path, user string) map[string]any {
	return map[string]any{
		FieldPath: path,
		FieldUser: user,
	}
}
