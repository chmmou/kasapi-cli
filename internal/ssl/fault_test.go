package ssl_test

import (
	"testing"

	"github.com/chmmou/kasapi-cli/internal/testutil"
)

// The ssl write slice is not implemented yet (#125), but its captured
// fault fixtures already live in testdata/ssl/. Anchoring them via the
// shared testutil.AssertFaultFixtures keeps them bound to the KAS
// contract instead of rotting unreferenced until the slice lands.
func TestFaultFixturesDecodeToDocumentedCodes(t *testing.T) {
	t.Parallel()
	testutil.AssertFaultFixtures(t, "ssl", map[string]string{
		"update_ssl_response_failed_nothing_to_do.xml": "nothing_to_do",
	})
}
