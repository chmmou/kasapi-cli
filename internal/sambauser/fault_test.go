package sambauser_test

import (
	"testing"

	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestFaultFixturesDecodeToDocumentedCodes(t *testing.T) {
	t.Parallel()
	testutil.AssertFaultFixtures(t, "sambauser", map[string]string{
		"add_sambauser_response_failed_couldnt_get_kas_ressources.xml":        "couldnt_get_kas_ressources",
		"add_sambauser_response_failed_max_sambauser_reached.xml":             "max_sambauser_reached",
		"add_sambauser_response_failed_missing_parameter.xml":                 "missing_parameter",
		"add_sambauser_response_failed_password_syntax_incorrect.xml":         "password_syntax_incorrect",
		"add_sambauser_response_failed_path_syntax_incorrect.xml":             "path_syntax_incorrect",
		"add_sambauser_response_failed_samba_comment_syntax_incorrect.xml":    "samba_comment_syntax_incorrect",
		"update_sambauser_response_failed_in_progress.xml":                    "in_progress",
		"update_sambauser_response_failed_missing_parameter.xml":              "missing_parameter",
		"update_sambauser_response_failed_nothing_to_do.xml":                  "nothing_to_do",
		"update_sambauser_response_failed_password_syntax_incorrect.xml":      "password_syntax_incorrect",
		"update_sambauser_response_failed_path_syntax_incorrect.xml":          "path_syntax_incorrect",
		"update_sambauser_response_failed_samba_comment_syntax_incorrect.xml": "samba_comment_syntax_incorrect",
		"update_sambauser_response_failed_samba_login_not_found.xml":          "samba_login_not_found",
		"update_sambauser_response_failed_samba_login_syntax_incorrect.xml":   "samba_login_syntax_incorrect",
		"delete_sambauser_response_failed_in_progress.xml":                    "in_progress",
		"delete_sambauser_response_failed_missing_parameter.xml":              "missing_parameter",
		"delete_sambauser_response_failed_samba_login_not_found.xml":          "samba_login_not_found",
		"delete_sambauser_response_failed_samba_login_syntax_incorrect.xml":   "samba_login_syntax_incorrect",
	})
}
