package api

import "github.com/chmmou/kasapi-cli/internal/soap"

// NewErrorForTest exposes the unexported newError constructor so the
// predicate tests in api_test can build *Error values directly without
// going through Call.
func NewErrorForTest(action, endpoint string, fe *soap.FaultError) *Error {
	return newError(action, endpoint, fe)
}
