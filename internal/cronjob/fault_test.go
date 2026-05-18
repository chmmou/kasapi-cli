package cronjob_test

import (
	"testing"

	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestFaultFixturesDecodeToDocumentedCodes(t *testing.T) {
	t.Parallel()
	testutil.AssertFaultFixtures(t, "cronjob", map[string]string{
		"add_cronjob_response_failed_cronjob_comment_syntax_incorrect.xml": "cronjob_comment_syntax_incorrect",
		"add_cronjob_response_failed_day_of_month_syntax_incorrect.xml":    "day_of_month_syntax_incorrect",
		"add_cronjob_response_failed_missing_parameter.xml":                "missing_parameter",
		"update_cronjob_response_failed_nothing_to_do.xml":                 "nothing_to_do",
		"update_cronjob_response_failed_cronjob_id_not_found.xml":          "cronjob_not_found",
		"delete_cronjob_response_failed_cronjob_id_not_found.xml":          "cronjob_id_not_found",
		"delete_cronjob_response_failed_missing_parameter.xml":             "missing_parameter",
	})
}
