package softwareinstall_test

import (
	"testing"

	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestFaultFixturesDecodeToDocumentedCodes(t *testing.T) {
	t.Parallel()
	testutil.AssertFaultFixtures(t, "softwareinstall", map[string]string{
		"add_softwareinstall_response_failed_admin_mail_syntax_incorrect.xml": "admin_mail_syntax_incorrect",
		"add_softwareinstall_response_failed_admin_password_forbidden.xml":    "admin_password_forbidden",
	})
}
