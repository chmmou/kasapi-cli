package ftpuser_test

import (
	"testing"

	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestFaultFixturesDecodeToDocumentedCodes(t *testing.T) {
	t.Parallel()
	testutil.AssertFaultFixtures(t, "ftpuser", map[string]string{
		"add_ftpuser_response_failed_ftp_comment_syntax_incorrect.xml": "ftp_comment_syntax_incorrect",
		"add_ftpuser_response_failed_ftp_path_syntax_incorrect.xml":    "ftp_path_syntax_incorrect",
	})
}
