package database_test

import (
	"testing"

	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestFaultFixturesDecodeToDocumentedCodes(t *testing.T) {
	t.Parallel()
	testutil.AssertFaultFixtures(t, "database", map[string]string{
		"update_database_response_failed_database_allowed_hosts_syntax_incorrect.xml": "database_allowed_hosts_syntax_incorrect",
		"update_database_response_failed_database_comment_syntax_incorrect.xml":       "database_comment_syntax_incorrect",
	})
}
