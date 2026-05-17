package ddns_test

import (
	"testing"

	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestFaultFixturesDecodeToDocumentedCodes(t *testing.T) {
	t.Parallel()
	testutil.AssertFaultFixtures(t, "ddns", map[string]string{
		"add_ddnsuser_response_failed_ddns_limit_reached.xml":       "ddns_limit_reached",
		"add_ddnsuser_response_failed_dns_settings_not_allowed.xml": "dns_settings_not_allowed",
	})
}
