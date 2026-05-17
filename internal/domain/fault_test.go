package domain_test

import (
	"testing"

	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestFaultFixturesDecodeToDocumentedCodes(t *testing.T) {
	t.Parallel()
	testutil.AssertFaultFixtures(t, "domain", map[string]string{
		"add_domain_response_failed_account_is_dummyaccount.xml":      "account_is_dummyaccount",
		"add_domain_response_failed_domain_path_syntax_incorrect.xml": "domain_path_syntax_incorrect",
	})
}
