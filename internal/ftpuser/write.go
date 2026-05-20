package ftpuser

import (
	"context"
	"errors"

	"github.com/chmmou/kasapi-cli/internal/kaswrite"
)

// ErrUnexpectedReturnString is the shared canonical post-call-contract
// sentinel, re-exported so errors.Is(err, ftpuser.ErrUnexpectedReturnString)
// keeps working and the slice stays self-describing. See
// kaswrite.ErrUnexpectedReturnString for the full contract.
var ErrUnexpectedReturnString = kaswrite.ErrUnexpectedReturnString

const (
	addAction    = "add_ftpuser"
	updateAction = "update_ftpuser"
	deleteAction = "delete_ftpuser"
)

// The Field-prefixed constants are the KAS request keys update_ftpuser
// accepts besides the ftp_login identifier; AddParams reuses the shared
// subset. Each is an optional wholesale replacement on update: only the
// keys the caller explicitly sets are sent, so an empty string is a
// meaningful value rather than "leave unchanged".
//
// The password key differs between actions: add_ftpuser sets the
// initial password via ftp_password, update_ftpuser replaces it via
// ftp_new_password. Both keys are kept explicit so neither path can
// silently send the wrong one (this asymmetry is in the KAS docs and
// the captured add_ftpuser / update_ftpuser request fixtures).
const (
	FieldLogin           = "ftp_login"
	FieldPassword        = "ftp_password"
	FieldNewPassword     = "ftp_new_password"
	FieldPath            = "ftp_path"
	FieldComment         = "ftp_comment"
	FieldPermissionRead  = "ftp_permission_read"
	FieldPermissionWrite = "ftp_permission_write"
	FieldPermissionList  = "ftp_permission_list"
	FieldVirusClamAV     = "ftp_virus_clamav"
)

// Spec carries the add_ftpuser request fields. Password and Comment are
// required by the KAS API and validated as non-empty before any SOAP
// call so the CLI can surface a fast validation error; the remaining
// fields are optional and sent verbatim (an empty value is sent as an
// empty string, matching the captured add_ftpuser request fixture).
//
// add_ftpuser takes no ftp_login: the server auto-generates the login
// and echoes it in ReturnInfo (e.g. "f0000004"), which Add returns.
type Spec struct {
	Password        string
	Comment         string
	Path            string
	PermissionRead  string
	PermissionWrite string
	PermissionList  string
	VirusClamAV     string
}

// Add creates an FTP user (add_ftpuser) and returns the login the
// server assigns, as echoed in ReturnInfo (e.g. "f0000004").
//
// Password and comment are validated before the SOAP call so an
// obviously incomplete spec fails fast; the remaining syntax validation
// (ftp_path_syntax_incorrect, password_syntax_incorrect, …) is left to
// the API and surfaces verbatim through the Caller.
func (cl *Client) Add(ctx context.Context, s Spec) (string, error) {
	switch {
	case s.Password == "":
		return "", errors.New("ftpuser: add_ftpuser requires a non-empty password")
	case s.Comment == "":
		return "", errors.New("ftpuser: add_ftpuser requires a non-empty comment")
	}
	resp, err := kaswrite.Call(ctx, cl.c, "ftpuser", addAction, AddParams(s))
	if err != nil {
		return "", err
	}
	return resp.Body.ReturnInfo.AsString(), nil
}

// AddParams builds the add_ftpuser KAS request parameter map. It is the
// single source of truth for the request shape so the CLI dry-run
// preview / audit record and the dispatched call cannot diverge. No
// ftp_login is sent — the server generates it.
func AddParams(s Spec) map[string]any {
	return map[string]any{
		FieldPassword:        s.Password,
		FieldComment:         s.Comment,
		FieldPath:            s.Path,
		FieldPermissionRead:  s.PermissionRead,
		FieldPermissionWrite: s.PermissionWrite,
		FieldPermissionList:  s.PermissionList,
		FieldVirusClamAV:     s.VirusClamAV,
	}
}

// Update changes one or more mutable fields of an existing FTP user
// (update_ftpuser). fields holds only the keys the caller wants to
// change (use the Field* constants; the password is FieldNewPassword
// here, not FieldPassword); each is applied wholesale. login and at
// least one field are required — update_ftpuser with nothing to change
// is rejected before the SOAP call (the API would fault nothing_to_do).
func (cl *Client) Update(ctx context.Context, login string, fields map[string]string) error {
	if login == "" {
		return errors.New("ftpuser: update_ftpuser requires a non-empty ftp login")
	}
	if len(fields) == 0 {
		return errors.New("ftpuser: update_ftpuser requires at least one field to change")
	}
	_, err := kaswrite.Call(ctx, cl.c, "ftpuser", updateAction, UpdateParams(login, fields))
	return err
}

// UpdateParams builds the update_ftpuser KAS request parameter map
// (single source of truth, see AddParams): the ftp_login identifier
// plus every caller-supplied mutable field verbatim.
func UpdateParams(login string, fields map[string]string) map[string]any {
	params := map[string]any{FieldLogin: login}
	for k, v := range fields {
		params[k] = v
	}
	return params
}

// Delete removes an FTP user (delete_ftpuser). A SOAP fault (e.g.
// ftp_login_not_found, in_progress) is surfaced verbatim by the Caller
// so the caller can classify it via the api error helpers.
func (cl *Client) Delete(ctx context.Context, login string) error {
	if login == "" {
		return errors.New("ftpuser: delete_ftpuser requires a non-empty ftp login")
	}
	_, err := kaswrite.Call(ctx, cl.c, "ftpuser", deleteAction, DeleteParams(login))
	return err
}

// DeleteParams builds the delete_ftpuser KAS request parameter map
// (single source of truth, see AddParams).
func DeleteParams(login string) map[string]any {
	return map[string]any{FieldLogin: login}
}
