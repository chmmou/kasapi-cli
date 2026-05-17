package sambauser_test

import (
	"testing"

	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestFaultFixturesDecodeToDocumentedCodes(t *testing.T) {
	t.Parallel()
	testutil.AssertFaultFixtures(t, "sambauser", map[string]string{
		"add_sambauser_response_failed_couldnt_get_kas_ressources.xml": "couldnt_get_kas_ressources",
		"add_sambauser_response_failed_max_sambauser_reached.xml":      "max_sambauser_reached",
	})
}
