package auth_test

import (
	"errors"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/auth"
)

// The positive-path coverage for IsOTPPinIncorrect runs through
// TestGetCredentialTokenOTPRejected in client_test.go (a real auth
// fault is produced by the fixture-backed httptest server, so the
// *auth.Error path through IsCode is exercised end-to-end).
//
// IsLoginFailed, IsLoginLocked, and IsUnknownSession do not yet have
// captured fault fixtures, so the positive-path tests will be added
// alongside the corresponding fixtures. The negative-path tests below
// pin the public sentinel API: any non-auth error (including nil) must
// not match.

func TestSentinelHelpersOnNonAuthError(t *testing.T) {
	t.Parallel()
	plain := errors.New("not an auth error")
	if auth.IsLoginFailed(plain) {
		t.Error("IsLoginFailed(plain) = true, want false")
	}
	if auth.IsLoginLocked(plain) {
		t.Error("IsLoginLocked(plain) = true, want false")
	}
	if auth.IsUnknownSession(plain) {
		t.Error("IsUnknownSession(plain) = true, want false")
	}
	if auth.IsOTPPinIncorrect(plain) {
		t.Error("IsOTPPinIncorrect(plain) = true, want false")
	}
	if got := auth.AsError(plain); got != nil {
		t.Errorf("AsError(plain) = %v, want nil", got)
	}
}

func TestSentinelHelpersOnNilError(t *testing.T) {
	t.Parallel()
	if auth.IsLoginFailed(nil) {
		t.Error("IsLoginFailed(nil) = true, want false")
	}
	if auth.IsLoginLocked(nil) {
		t.Error("IsLoginLocked(nil) = true, want false")
	}
	if auth.IsUnknownSession(nil) {
		t.Error("IsUnknownSession(nil) = true, want false")
	}
	if got := auth.AsError(nil); got != nil {
		t.Errorf("AsError(nil) = %v, want nil", got)
	}
}

func TestIsCodeOnUnknownCode(t *testing.T) {
	t.Parallel()
	if auth.IsCode(errors.New("boom"), auth.CodeLoginFailed) {
		t.Error("IsCode on plain error matched a known code")
	}
}
