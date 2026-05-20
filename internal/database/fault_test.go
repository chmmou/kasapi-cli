package database_test

import (
	"testing"

	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestFaultFixturesDecodeToDocumentedCodes(t *testing.T) {
	t.Parallel()
	testutil.AssertFaultFixtures(t, "database", map[string]string{
		// add_database faults
		"add_database_response_failed_account_is_dummyaccount.xml":                 "account_is_dummyaccount",
		"add_database_response_failed_cant_connect_to_mysql_on_this_server.xml":    "cant_connect_to_mysql_on_this_server",
		"add_database_response_failed_couldnt_get_kas_ressources.xml":              "couldnt_get_kas_ressources",
		"add_database_response_failed_database_allowed_hosts_syntax_incorrect.xml": "database_allowed_hosts_syntax_incorrect",
		"add_database_response_failed_database_comment_syntax_incorrect.xml":       "database_comment_syntax_incorrect",
		"add_database_response_failed_max_database_reached.xml":                    "max_database_reached",
		"add_database_response_failed_missing_parameter.xml":                       "missing_parameter",
		"add_database_response_failed_no_mysql_on_this_server.xml":                 "no_mysql_on_this_server",
		"add_database_response_failed_password_syntax_incorrect.xml":               "password_syntax_incorrect",
		// update_database faults
		"update_database_response_failed_database_allowed_hosts_syntax_incorrect.xml": "database_allowed_hosts_syntax_incorrect",
		"update_database_response_failed_database_comment_syntax_incorrect.xml":       "database_comment_syntax_incorrect",
		"update_database_response_failed_database_login_not_found.xml":                "database_login_not_found",
		"update_database_response_failed_missing_parameter.xml":                       "missing_parameter",
		"update_database_response_failed_password_syntax_incorrect.xml":               "password_syntax_incorrect",
		// delete_database faults
		"delete_database_response_failed_cant_connect_to_mysql_on_this_server.xml": "cant_connect_to_mysql_on_this_server",
		"delete_database_response_failed_database_login_not_found.xml":             "database_login_not_found",
		"delete_database_response_failed_in_progress.xml":                          "in_progress",
		"delete_database_response_failed_missing_parameter.xml":                    "missing_parameter",
		"delete_database_response_failed_no_mysql_on_this_server.xml":              "no_mysql_on_this_server",
	})
}
