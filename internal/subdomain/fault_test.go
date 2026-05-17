package subdomain_test

import (
	"testing"

	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestFaultFixturesDecodeToDocumentedCodes(t *testing.T) {
	t.Parallel()
	testutil.AssertFaultFixtures(t, "subdomain", map[string]string{
		"add_subdomain_response_failed_account_is_dummyaccount.xml":                "account_is_dummyaccount",
		"add_subdomain_response_failed_domain_for_this_subdomain_doesnt_exist.xml": "domain_for_this_subdomain_doesnt_exist",
	})
}
