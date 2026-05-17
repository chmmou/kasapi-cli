package mailaccount_test

import (
	"testing"

	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestFaultFixturesDecodeToDocumentedCodes(t *testing.T) {
	t.Parallel()
	testutil.AssertFaultFixtures(t, "mailaccount", map[string]string{
		"add_mailaccount_response_failed_copy_adress_like_mailaccount.xml": "copy_adress_like_mailaccount",
		"add_mailaccount_response_failed_copy_adress_syntax_incorrect.xml": "copy_adress_syntax_incorrect",
	})
}
