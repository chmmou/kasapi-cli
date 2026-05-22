package mailaccount

import (
	"context"
	"errors"

	"github.com/chmmou/kasapi-cli/internal/kaswrite"
)

// ErrUnexpectedReturnString is the shared canonical post-call-contract
// sentinel, re-exported so errors.Is(err, mailaccount.ErrUnexpectedReturnString)
// keeps working and the slice stays self-describing. See
// kaswrite.ErrUnexpectedReturnString for the full contract.
var ErrUnexpectedReturnString = kaswrite.ErrUnexpectedReturnString

const (
	addAction    = "add_mailaccount"
	updateAction = "update_mailaccount"
	deleteAction = "delete_mailaccount"
)

// The Field-prefixed constants are the KAS request keys
// update_mailaccount accepts besides the mail_login identifier; each is
// an optional wholesale replacement, so only the keys the caller
// explicitly sets are sent and an empty string is a meaningful value
// rather than "leave unchanged".
//
// Two asymmetries against the add path are baked into the wire keys and
// must not be conflated:
//
//   - The password key differs by action: add_mailaccount sets the
//     initial password via mail_password, update_mailaccount replaces it
//     via mail_new_password (the same _new_password split as
//     ftpuser/sambauser/database). FieldNewPassword is the update key;
//     add uses its own mail_password key in AddParams.
//   - FieldIsActive (is_active) and FieldLogin (mail_login) exist only on
//     update: add_mailaccount takes no mail_login (the server generates
//     it) and has no is_active toggle (a freshly created account is
//     active).
//
// FieldCopyAddress keeps the legacy single-d "copy_adress" spelling
// because that is the literal wire key both add and update accept
// (mirrors the mail_adresses/mail_copy_adress legacy keys on the read
// path).
const (
	FieldLogin                = "mail_login"
	FieldNewPassword          = "mail_new_password"
	FieldWebmailAutologin     = "webmail_autologin"
	FieldResponder            = "responder"
	FieldResponderContentType = "mail_responder_content_type"
	FieldResponderDisplayName = "mail_responder_displayname"
	FieldResponderText        = "responder_text"
	FieldCopyAddress          = "copy_adress"
	FieldIsActive             = "is_active"
	FieldSenderAlias          = "mail_sender_alias"
	FieldXListEnabled         = "mail_xlist_enabled"
	FieldXListSent            = "mail_xlist_sent"
	FieldXListDrafts          = "mail_xlist_drafts"
	FieldXListTrash           = "mail_xlist_trash"
	FieldXListSpam            = "mail_xlist_spam"
	FieldXListArchiv          = "mail_xlist_archiv"
	FieldAllowNets            = "mail_allow_nets"
)

// Spec carries the add_mailaccount request fields. LocalPart, DomainPart
// and Password are required by the KAS API and validated as non-empty
// before any SOAP call so the CLI can surface a fast validation error;
// the remaining fields are optional and sent verbatim (an empty value is
// sent as an empty string, matching the captured add_mailaccount request
// fixture, which carries every key even when blank).
//
// add_mailaccount takes no mail_login: the server auto-generates the
// login and echoes it in ReturnInfo (e.g. "m0000001"), which Add
// returns. Responder is passed through verbatim — the API accepts the
// literal strings "N" / "Y" or a "<start>|<end>" timestamp range.
type Spec struct {
	LocalPart            string
	DomainPart           string
	Password             string
	WebmailAutologin     string
	Responder            string
	ResponderContentType string
	ResponderDisplayName string
	ResponderText        string
	CopyAddress          string
	SenderAlias          string
	XListEnabled         string
	XListSent            string
	XListDrafts          string
	XListTrash           string
	XListSpam            string
	XListArchiv          string
	AllowNets            string
}

// Add creates a mail account (add_mailaccount) and returns the login the
// server assigns, as echoed in ReturnInfo (e.g. "m0000001").
//
// LocalPart, DomainPart and Password are validated before the SOAP call
// so an obviously incomplete spec fails fast; the remaining syntax
// validation (email_syntax_incorrect, password_syntax_incorrect,
// mail_xlist_*_syntax_incorrect, responder_*, …) is left to the API and
// surfaces verbatim through the Caller.
func (cl *Client) Add(ctx context.Context, s Spec) (string, error) {
	switch {
	case s.LocalPart == "":
		return "", errors.New("mailaccount: add_mailaccount requires a non-empty local part")
	case s.DomainPart == "":
		return "", errors.New("mailaccount: add_mailaccount requires a non-empty domain part")
	case s.Password == "":
		return "", errors.New("mailaccount: add_mailaccount requires a non-empty password")
	}
	resp, err := kaswrite.Call(ctx, cl.c, "mailaccount", addAction, AddParams(s))
	if err != nil {
		return "", err
	}
	return resp.Body.ReturnInfo.AsString(), nil
}

// AddParams builds the add_mailaccount KAS request parameter map. It is
// the single source of truth for the request shape so the CLI dry-run
// preview / audit record and the dispatched call cannot diverge. No
// mail_login is sent — the server generates it. The address is sent
// decomposed into local_part / domain_part (add_mailaccount's keys);
// every optional field is sent verbatim, matching the captured request
// fixture.
func AddParams(s Spec) map[string]any {
	return map[string]any{
		"local_part":              s.LocalPart,
		"domain_part":             s.DomainPart,
		"mail_password":           s.Password,
		FieldWebmailAutologin:     s.WebmailAutologin,
		FieldResponder:            s.Responder,
		FieldResponderContentType: s.ResponderContentType,
		FieldResponderDisplayName: s.ResponderDisplayName,
		FieldResponderText:        s.ResponderText,
		FieldCopyAddress:          s.CopyAddress,
		FieldSenderAlias:          s.SenderAlias,
		FieldXListEnabled:         s.XListEnabled,
		FieldXListSent:            s.XListSent,
		FieldXListDrafts:          s.XListDrafts,
		FieldXListTrash:           s.XListTrash,
		FieldXListSpam:            s.XListSpam,
		FieldXListArchiv:          s.XListArchiv,
		FieldAllowNets:            s.AllowNets,
	}
}

// Update changes one or more mutable fields of an existing mail account
// (update_mailaccount). fields holds only the keys the caller wants to
// change (use the Field* constants; the password is FieldNewPassword
// here, not the add-only mail_password); each is applied wholesale.
// login and at least one field are required — update_mailaccount with
// nothing to change is rejected before the SOAP call (the API would
// fault nothing_to_do).
func (cl *Client) Update(ctx context.Context, login string, fields map[string]string) error {
	if login == "" {
		return errors.New("mailaccount: update_mailaccount requires a non-empty mail login")
	}
	if len(fields) == 0 {
		return errors.New("mailaccount: update_mailaccount requires at least one field to change")
	}
	_, err := kaswrite.Call(ctx, cl.c, "mailaccount", updateAction, UpdateParams(login, fields))
	return err
}

// UpdateParams builds the update_mailaccount KAS request parameter map
// (single source of truth, see AddParams): the mail_login identifier
// plus every caller-supplied mutable field verbatim.
func UpdateParams(login string, fields map[string]string) map[string]any {
	params := map[string]any{FieldLogin: login}
	for k, v := range fields {
		params[k] = v
	}
	return params
}

// Delete removes a mail account (delete_mailaccount). A SOAP fault (e.g.
// mail_login_not_found, in_progress, mail_loop_detected) is surfaced
// verbatim by the Caller so the caller can classify it via the api error
// helpers.
func (cl *Client) Delete(ctx context.Context, login string) error {
	if login == "" {
		return errors.New("mailaccount: delete_mailaccount requires a non-empty mail login")
	}
	_, err := kaswrite.Call(ctx, cl.c, "mailaccount", deleteAction, DeleteParams(login))
	return err
}

// DeleteParams builds the delete_mailaccount KAS request parameter map
// (single source of truth, see AddParams).
func DeleteParams(login string) map[string]any {
	return map[string]any{FieldLogin: login}
}
