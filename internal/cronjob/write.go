package cronjob

import (
	"context"
	"errors"

	"github.com/chmmou/kasapi-cli/internal/kaswrite"
)

// ErrUnexpectedReturnString is the shared canonical post-call-contract
// sentinel, re-exported so errors.Is(err, cronjob.ErrUnexpectedReturnString)
// keeps working and the slice stays self-describing. See
// kaswrite.ErrUnexpectedReturnString for the full contract.
var ErrUnexpectedReturnString = kaswrite.ErrUnexpectedReturnString

const (
	addAction    = "add_cronjob"
	updateAction = "update_cronjob"
	deleteAction = "delete_cronjob"
)

// The Field-prefixed constants are the KAS request keys update_cronjob
// accepts besides the cronjob_id identifier; AddParams uses the same
// keys. Each is an optional wholesale replacement on update: only the
// keys the caller explicitly sets are sent, so an empty string is a
// meaningful value rather than "leave unchanged".
//
// FieldMailAdress mirrors the KAS wire key, which the API spells with a
// single 'd' (mail_adress) — the same quirk the read-side Cronjob
// struct documents. Both the add_cronjob and update_cronjob request
// fixtures use the single-d spelling, so it is the authoritative
// request key even though the documentation's parameter table is
// inconsistent.
const (
	FieldProtocol      = "protocol"
	FieldHTTPURL       = "http_url"
	FieldComment       = "cronjob_comment"
	FieldMinute        = "minute"
	FieldHour          = "hour"
	FieldDayOfMonth    = "day_of_month"
	FieldMonth         = "month"
	FieldDayOfWeek     = "day_of_week"
	FieldHTTPUser      = "http_user"
	FieldHTTPPassword  = "http_password"
	FieldMailAdress    = "mail_adress"
	FieldMailCondition = "mail_condition"
	FieldMailSubject   = "mail_subject"
	FieldIsActive      = "is_active"
)

// Spec carries the add_cronjob request fields. Protocol, HTTPURL,
// Comment and the five schedule fields are required by the KAS API and
// validated as non-empty before any SOAP call so the CLI can surface a
// fast validation error; the remaining fields are optional and sent
// verbatim (an empty value is sent as an empty string, matching the
// captured add_cronjob request fixture).
type Spec struct {
	Protocol      string
	HTTPURL       string
	Comment       string
	Minute        string
	Hour          string
	DayOfMonth    string
	Month         string
	DayOfWeek     string
	HTTPUser      string
	HTTPPassword  string
	MailAdress    string
	MailCondition string
	MailSubject   string
	IsActive      string
}

// Add creates a cronjob (add_cronjob) and returns the numeric cronjob
// id the server assigns, as echoed in ReturnInfo (e.g. "324700").
//
// The eight KAS-required fields are validated before the SOAP call so
// an obviously incomplete spec fails fast; the remaining schedule /
// syntax validation is left to the API, whose documented faults
// (minute_syntax_incorrect, time_not_allowed, …) surface verbatim
// through the Caller.
func (cl *Client) Add(ctx context.Context, s Spec) (string, error) {
	switch {
	case s.Protocol == "":
		return "", errors.New("cronjob: add_cronjob requires a non-empty protocol")
	case s.HTTPURL == "":
		return "", errors.New("cronjob: add_cronjob requires a non-empty http url")
	case s.Comment == "":
		return "", errors.New("cronjob: add_cronjob requires a non-empty comment")
	case s.Minute == "":
		return "", errors.New("cronjob: add_cronjob requires a non-empty minute schedule")
	case s.Hour == "":
		return "", errors.New("cronjob: add_cronjob requires a non-empty hour schedule")
	case s.DayOfMonth == "":
		return "", errors.New("cronjob: add_cronjob requires a non-empty day_of_month schedule")
	case s.Month == "":
		return "", errors.New("cronjob: add_cronjob requires a non-empty month schedule")
	case s.DayOfWeek == "":
		return "", errors.New("cronjob: add_cronjob requires a non-empty day_of_week schedule")
	}
	resp, err := kaswrite.Call(ctx, cl.c, "cronjob", addAction, AddParams(s))
	if err != nil {
		return "", err
	}
	return resp.Body.ReturnInfo.AsString(), nil
}

// AddParams builds the add_cronjob KAS request parameter map. It is the
// single source of truth for the request shape so the CLI dry-run
// preview / audit record and the dispatched call cannot diverge.
func AddParams(s Spec) map[string]any {
	return map[string]any{
		FieldProtocol:      s.Protocol,
		FieldHTTPURL:       s.HTTPURL,
		FieldComment:       s.Comment,
		FieldMinute:        s.Minute,
		FieldHour:          s.Hour,
		FieldDayOfMonth:    s.DayOfMonth,
		FieldMonth:         s.Month,
		FieldDayOfWeek:     s.DayOfWeek,
		FieldHTTPUser:      s.HTTPUser,
		FieldHTTPPassword:  s.HTTPPassword,
		FieldMailAdress:    s.MailAdress,
		FieldMailCondition: s.MailCondition,
		FieldMailSubject:   s.MailSubject,
		FieldIsActive:      s.IsActive,
	}
}

// Update changes one or more mutable fields of an existing cronjob
// (update_cronjob). fields holds only the keys the caller wants to
// change (use the Field* constants); each is applied wholesale. id and
// at least one field are required — update_cronjob with nothing to
// change is rejected before the SOAP call (the API would fault
// nothing_to_do).
func (cl *Client) Update(ctx context.Context, id string, fields map[string]string) error {
	if id == "" {
		return errors.New("cronjob: update_cronjob requires a non-empty cronjob id")
	}
	if len(fields) == 0 {
		return errors.New("cronjob: update_cronjob requires at least one field to change")
	}
	_, err := kaswrite.Call(ctx, cl.c, "cronjob", updateAction, UpdateParams(id, fields))
	return err
}

// UpdateParams builds the update_cronjob KAS request parameter map
// (single source of truth, see AddParams): the cronjob_id identifier
// plus every caller-supplied mutable field verbatim.
func UpdateParams(id string, fields map[string]string) map[string]any {
	params := map[string]any{"cronjob_id": id}
	for k, v := range fields {
		params[k] = v
	}
	return params
}

// Delete removes a cronjob (delete_cronjob). A SOAP fault (e.g.
// cronjob_id_not_found, in_progress) is surfaced verbatim by the Caller
// so the caller can classify it via the api error helpers.
func (cl *Client) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("cronjob: delete_cronjob requires a non-empty cronjob id")
	}
	_, err := kaswrite.Call(ctx, cl.c, "cronjob", deleteAction, DeleteParams(id))
	return err
}

// DeleteParams builds the delete_cronjob KAS request parameter map
// (single source of truth, see AddParams).
func DeleteParams(id string) map[string]any {
	return map[string]any{"cronjob_id": id}
}
