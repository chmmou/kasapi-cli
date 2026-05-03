package auth

import (
	"errors"
	"fmt"

	"github.com/chmmou/kasapi-cli/internal/soap"
)

// Stable KasAuth fault codes that callers commonly branch on. The full
// set of codes is open; new ones surface verbatim in Error.Code.
const (
	CodeLoginFailed     = "kas_login_failed"
	CodeLoginLocked     = "kas_login_locked"
	CodeOTPPinIncorrect = "otp_pin_incorrect"
	CodeUnknownSession  = "unknown_session"
	CodeGotNoLoginData  = "got_no_login_data"
)

// Error is the typed wrapper for a SOAP-ENV:Fault returned by KasAuth.
// Login identifies the credential whose authentication failed; Code is
// the stable KAS error string.
type Error struct {
	Code    string
	Message string
	Login   string

	fault *soap.FaultError
}

func (e *Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("kas-auth %s: %s: %s", e.Login, e.Code, e.Message)
	}
	return fmt.Sprintf("kas-auth %s: %s", e.Login, e.Code)
}

// Unwrap returns the underlying *soap.FaultError so errors.As can
// recover the SOAP-level error if needed.
func (e *Error) Unwrap() error { return e.fault }

func newError(login string, fe *soap.FaultError) *Error {
	return &Error{
		Code:    fe.Fault.String,
		Message: fe.Fault.Detail,
		Login:   login,
		fault:   fe,
	}
}

func codeOf(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

// IsCode reports whether err is an *Error with the given code.
func IsCode(err error, code string) bool { return codeOf(err) == code }

// IsLoginFailed reports whether err is the kas_login_failed fault.
func IsLoginFailed(err error) bool { return IsCode(err, CodeLoginFailed) }

// IsLoginLocked reports whether err is the kas_login_locked fault.
func IsLoginLocked(err error) bool { return IsCode(err, CodeLoginLocked) }

// IsOTPPinIncorrect reports whether err is the otp_pin_incorrect fault.
func IsOTPPinIncorrect(err error) bool { return IsCode(err, CodeOTPPinIncorrect) }

// IsUnknownSession reports whether err is the unknown_session fault,
// returned when authenticating with kas_auth_type=session against a
// stale or revoked token.
func IsUnknownSession(err error) bool { return IsCode(err, CodeUnknownSession) }

// AsError extracts an *Error from err, returning nil if err is not an
// auth fault.
func AsError(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return nil
}
