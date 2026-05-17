package directoryprotection_test

import (
	"testing"

	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestFaultFixturesDecodeToDocumentedCodes(t *testing.T) {
	t.Parallel()
	testutil.AssertFaultFixtures(t, "directoryprotection", map[string]string{
		"add_directoryprotection_response_failed_directory_authname_syntax_incorrect.xml": "directory_authname_syntax_incorrect",
		"add_directoryprotection_response_failed_directory_password_syntax_incorrect.xml": "directory_password_syntax_incorrect",
	})
}
