package symlink_test

import (
	"testing"

	"github.com/chmmou/kasapi-cli/internal/testutil"
)

// The symlink write slice is not implemented yet (#125), but its
// captured fault fixtures already live in testdata/symlink/. Anchoring
// them via the shared testutil.AssertFaultFixtures keeps them bound to
// the KAS contract instead of rotting unreferenced until the slice
// lands.
func TestFaultFixturesDecodeToDocumentedCodes(t *testing.T) {
	t.Parallel()
	testutil.AssertFaultFixtures(t, "symlink", map[string]string{
		"add_symlink_response_failed_in_progress.xml": "in_progress",
	})
}
