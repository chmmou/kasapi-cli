package dns_test

import (
	"testing"

	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestFaultFixturesDecodeToDocumentedCodes(t *testing.T) {
	t.Parallel()
	testutil.AssertFaultFixtures(t, "dns", map[string]string{
		"add_dns_settings_response_failed_dns_settings_not_allowed.xml": "dns_settings_not_allowed",
		"add_dns_settings_response_failed_missing_parameter.xml":        "missing_parameter",
	})
}
