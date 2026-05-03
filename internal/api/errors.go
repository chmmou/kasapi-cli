package api

import (
	"errors"
	"fmt"
	"strings"

	"github.com/chmmou/kasapi-cli/internal/soap"
)

// Stable KAS error codes referenced by name elsewhere in the package and
// by callers. The full set of codes is open; new ones surface verbatim
// in Error.Code.
const (
	CodeNoAuth           = "no_auth"
	CodeAccessForbidden  = "kas_access_forbidden"
	CodeNoAction         = "no_action"
	CodeUnknownAction    = "unkown_action"
	CodeGotNoLoginData   = "got_no_login_data"
	CodeUnknownSession   = "unknown_session"
	CodeFloodProtection  = "flood_protection"
	CodeInProgress       = "in_progress"
	CodeMissingParameter = "missing_parameter"
	CodeNothingToDo      = "nothing_to_do"
	CodeEmptyList        = "empty_list"
)

// Error is the typed wrapper for a SOAP-ENV:Fault body returned by the
// KAS API. Code is the stable error string from <faultstring>; Message
// carries the optional <detail> text. Action and Endpoint identify the
// call that produced the fault.
type Error struct {
	Code     string
	Message  string
	Action   string
	Endpoint string

	fault *soap.FaultError
}

func (e *Error) Error() string {
	parts := []string{fmt.Sprintf("kas %s: %s", e.Action, e.Code)}
	if e.Message != "" {
		parts = append(parts, e.Message)
	}
	return strings.Join(parts, ": ")
}

// Unwrap exposes the underlying *soap.FaultError so errors.As can recover
// the original SOAP-level error if needed.
func (e *Error) Unwrap() error { return e.fault }

func newError(action, endpoint string, fe *soap.FaultError) *Error {
	return &Error{
		Code:     fe.Fault.String,
		Message:  fe.Fault.Detail,
		Action:   action,
		Endpoint: endpoint,
		fault:    fe,
	}
}

// codeOf extracts the KAS error code from err. Returns "" if err is not
// an *Error.
func codeOf(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

// IsCode reports whether err is an *Error with the given code.
func IsCode(err error, code string) bool { return codeOf(err) == code }

// IsAuthFailure reports whether err is an authentication failure that a
// session-token client should respond to by re-authenticating. This
// includes no_auth, unknown_session, kas_access_forbidden, and the
// generic got_no_login_data envelope returned when the request lacked
// kas_login.
func IsAuthFailure(err error) bool {
	switch codeOf(err) {
	case CodeNoAuth, CodeUnknownSession, CodeAccessForbidden, CodeGotNoLoginData:
		return true
	}
	return false
}

// IsFloodProtection reports whether err is the flood_protection fault.
// Callers that observe this should back off; the transport gate alone
// cannot prevent it because the fault carries no KasFloodDelay value.
func IsFloodProtection(err error) bool { return IsCode(err, CodeFloodProtection) }

// IsInProgress reports whether the fault is in_progress (a previous
// asynchronous operation on the same target has not finished yet).
func IsInProgress(err error) bool { return IsCode(err, CodeInProgress) }

// IsMissingParameter reports whether err is missing_parameter.
func IsMissingParameter(err error) bool { return IsCode(err, CodeMissingParameter) }

// IsNothingToDo reports whether err is nothing_to_do (an update with no
// actual changes).
func IsNothingToDo(err error) bool { return IsCode(err, CodeNothingToDo) }

// IsNotFound reports whether the fault code names a missing entity. The
// KAS API uses several suffixes for this; the predicate covers all of
// them: *_not_found, *_doesnt_exist, *_doenst_exist (server typo).
func IsNotFound(err error) bool {
	c := codeOf(err)
	if c == "" {
		return false
	}
	return strings.HasSuffix(c, "_not_found") ||
		strings.HasSuffix(c, "_not_found_in_kas") ||
		strings.HasSuffix(c, "_doesnt_exist") ||
		strings.HasSuffix(c, "_doenst_exist")
}

// IsSyntaxError reports whether the fault code is a *_syntax_incorrect
// validation error.
func IsSyntaxError(err error) bool {
	c := codeOf(err)
	return c != "" && strings.HasSuffix(c, "_syntax_incorrect")
}

// IsMaxReached reports whether the fault code is a max_*_reached or a
// generic *_limit_reached quota error.
func IsMaxReached(err error) bool {
	c := codeOf(err)
	if c == "" {
		return false
	}
	return (strings.HasPrefix(c, "max_") && strings.HasSuffix(c, "_reached")) ||
		strings.HasSuffix(c, "_limit_reached")
}

// AsError extracts an *Error from err, returning nil if err is not an
// api fault. Convenience wrapper around errors.As for callers that want
// to inspect Code or Message directly.
func AsError(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return nil
}
