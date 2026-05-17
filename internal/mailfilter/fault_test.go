package mailfilter_test

import (
	"testing"

	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestFaultFixturesDecodeToDocumentedCodes(t *testing.T) {
	t.Parallel()
	testutil.AssertFaultFixtures(t, "mailfilter", map[string]string{
		"add_mailstandardfilter_response_failed_action_syntax_incorrect.xml": "action_syntax_incorrect",
		"add_mailstandardfilter_response_failed_filter_doesnt_exist.xml":     "filter_doesnt_exist",
	})
}
