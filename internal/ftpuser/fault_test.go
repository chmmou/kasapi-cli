package ftpuser_test

import (
	"testing"

	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestFaultFixturesDecodeToDocumentedCodes(t *testing.T) {
	t.Parallel()
	testutil.AssertFaultFixtures(t, "ftpuser", map[string]string{
		"add_ftpuser_response_failed_ftp_comment_syntax_incorrect.xml":             "ftp_comment_syntax_incorrect",
		"add_ftpuser_response_failed_ftp_path_syntax_incorrect.xml":                "ftp_path_syntax_incorrect",
		"add_ftpuser_response_failed_ftp_permission_list_syntax_incorrect.xml":     "ftp_permission_list_syntax_incorrect",
		"add_ftpuser_response_failed_ftp_permission_read_syntax_incorrect.xml":     "ftp_permission_read_syntax_incorrect",
		"add_ftpuser_response_failed_ftp_permission_write_syntax_incorrect.xml":    "ftp_permission_write_syntax_incorrect",
		"add_ftpuser_response_failed_ftp_virus_clamav_syntax_incorrect.xml":        "ftp_virus_clamav_syntax_incorrect",
		"add_ftpuser_response_failed_max_ftpuser_reached.xml":                      "max_ftpuser_reached",
		"add_ftpuser_response_failed_missing_parameter.xml":                        "missing_parameter",
		"add_ftpuser_response_failed_password_syntax_incorrect.xml":                "password_syntax_incorrect",
		"update_ftpuser_response_failed_ftp_comment_syntax_incorrect.xml":          "ftp_comment_syntax_incorrect",
		"update_ftpuser_response_failed_ftp_login_not_found.xml":                   "ftp_login_not_found",
		"update_ftpuser_response_failed_ftp_login_syntax_incorrect.xml":            "ftp_login_syntax_incorrect",
		"update_ftpuser_response_failed_ftp_path_syntax_incorrect.xml":             "ftp_path_syntax_incorrect",
		"update_ftpuser_response_failed_ftp_permission_list_syntax_incorrect.xml":  "ftp_permission_list_syntax_incorrect",
		"update_ftpuser_response_failed_ftp_permission_read_syntax_incorrect.xml":  "ftp_permission_read_syntax_incorrect",
		"update_ftpuser_response_failed_ftp_permission_write_syntax_incorrect.xml": "ftp_permission_write_syntax_incorrect",
		"update_ftpuser_response_failed_ftp_virus_clamav_syntax_incorrect.xml":     "ftp_virus_clamav_syntax_incorrect",
		"update_ftpuser_response_failed_nothing_to_do.xml":                         "nothing_to_do",
		"update_ftpuser_response_failed_password_syntax_incorrect.xml":             "password_syntax_incorrect",
		"delete_ftpuser_response_failed_ftp_login_not_found.xml":                   "ftp_login_not_found",
		"delete_ftpuser_response_failed_ftp_login_syntax_incorrect.xml":            "ftp_login_syntax_incorrect",
		"delete_ftpuser_response_failed_missing_parameter.xml":                     "missing_parameter",
	})
}
