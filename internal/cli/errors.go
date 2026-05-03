package cli

import (
	"errors"
	"fmt"
)

// Exit codes used by the kasapi-cli binary. They follow the convention
// declared in issue #12: 0 success, 1 user error (bad flags, bad config,
// missing file), 2 API error (KAS fault, network failure).
const (
	ExitOK        = 0
	ExitUserError = 1
	ExitAPIError  = 2
)

// ExitError carries an exit code alongside the underlying error so the
// binary entry point can translate failures to the documented codes.
// Use UserError or APIError to construct one; falling through with a
// raw error from a subcommand maps to ExitAPIError by convention.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string { return e.Err.Error() }

func (e *ExitError) Unwrap() error { return e.Err }

// UserError wraps err as an ExitError with ExitUserError. msg is
// optional; when non-empty it is prepended to the underlying message.
func UserError(err error, msg string) *ExitError {
	return &ExitError{Code: ExitUserError, Err: prefix(msg, err)}
}

// APIError wraps err as an ExitError with ExitAPIError.
func APIError(err error, msg string) *ExitError {
	return &ExitError{Code: ExitAPIError, Err: prefix(msg, err)}
}

// CodeFor returns the documented exit code for err. *ExitError values
// carry their own code; any other non-nil error defaults to ExitAPIError
// since user-input validation should always produce an *ExitError.
func CodeFor(err error) int {
	if err == nil {
		return ExitOK
	}
	var ee *ExitError
	if errors.As(err, &ee) {
		return ee.Code
	}
	return ExitAPIError
}

func prefix(msg string, err error) error {
	if msg == "" {
		return err
	}
	return fmt.Errorf("%s: %w", msg, err)
}
