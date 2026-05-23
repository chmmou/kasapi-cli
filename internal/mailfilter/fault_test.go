package mailfilter_test

import (
	"testing"

	"github.com/chmmou/kasapi-cli/internal/testutil"
)

// TestFaultFixturesDecodeToDocumentedCodes is the
// testutil.AssertFaultFixtures anchor for the mailfilter slice: every
// _response_failed_*.xml under testdata/mailfilter/ must decode to a
// *soap.FaultError with a non-empty code, and each entry below pins the
// captured faultstring to the slug encoded in the filename. The
// envelope-level internal-server-error captured for the
// delete_mailstandardfilter quirk lives at the shared top level
// (testdata/response_failed_internal_server_error.xml) and is exercised
// by write_test.go and by the soap-level fixture walker.
func TestFaultFixturesDecodeToDocumentedCodes(t *testing.T) {
	t.Parallel()
	testutil.AssertFaultFixtures(t, "mailfilter", map[string]string{
		"add_mailstandardfilter_response_failed_action_syntax_incorrect.xml":        "action_syntax_incorrect",
		"add_mailstandardfilter_response_failed_filter_doesnt_exist.xml":            "filter_doesnt_exist",
		"add_mailstandardfilter_response_failed_filter_not_allowed.xml":             "filter_not_allowed",
		"add_mailstandardfilter_response_failed_login_not_found.xml":                "login_not_found",
		"add_mailstandardfilter_response_failed_mail_login_syntax_incorrect.xml":    "mail_login_syntax_incorrect",
		"add_mailstandardfilter_response_failed_missing_parameter.xml":              "missing_parameter",
		"add_mailstandardfilter_response_failed_spamfilter_not_in_contract.xml":     "spamfilter_not_in_contract",
		"add_mailstandardfilter_response_failed_unknown_filtertype.xml":             "unknown_filtertype",
		"delete_mailstandardfilter_response_failed_in_progress.xml":                 "in_progress",
		"delete_mailstandardfilter_response_failed_login_not_found.xml":             "login_not_found",
		"delete_mailstandardfilter_response_failed_mail_login_syntax_incorrect.xml": "mail_login_syntax_incorrect",
		"delete_mailstandardfilter_response_failed_missing_parameter.xml":           "missing_parameter",
	})
}
