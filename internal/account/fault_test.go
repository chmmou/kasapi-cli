package account_test

import (
	"testing"

	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestFaultFixturesDecodeToDocumentedCodes(t *testing.T) {
	t.Parallel()
	testutil.AssertFaultFixtures(t, "account", map[string]string{
		"add_account_response_failed_account_comment_syntax_incorrect.xml":      "account_comment_syntax_incorrect",
		"add_account_response_failed_account_contact_mail_syntax_incorrect.xml": "account_contact_mail_syntax_incorrect",
	})
}
