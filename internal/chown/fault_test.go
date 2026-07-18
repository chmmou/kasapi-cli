package chown_test

import (
	"testing"

	"github.com/chmmou/kasapi-cli/internal/testutil"
)

// The chown write slice is not implemented yet (#125), but its captured
// fault fixtures already live in testdata/chown/. Anchoring them via the
// shared testutil.AssertFaultFixtures keeps them bound to the KAS
// contract instead of rotting unreferenced until the slice lands.
func TestFaultFixturesDecodeToDocumentedCodes(t *testing.T) {
	t.Parallel()
	testutil.AssertFaultFixtures(t, "chown", map[string]string{
		"update_chown_response_failed_missing_parameter.xml": "missing_parameter",
	})
}
