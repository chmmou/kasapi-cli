package session_test

import (
	"testing"

	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestFaultFixturesDecodeToDocumentedCodes(t *testing.T) {
	t.Parallel()
	testutil.AssertFaultFixtures(t, "session", map[string]string{
		"add_session_response_failed_otp_pin_incorrect.xml":  "otp_pin_incorrect",
		"delete_session_response_failed_unknown_session.xml": "unknown_session",
	})
}
