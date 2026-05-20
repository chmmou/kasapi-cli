package ddns_test

import (
	"testing"

	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestFaultFixturesDecodeToDocumentedCodes(t *testing.T) {
	t.Parallel()
	testutil.AssertFaultFixtures(t, "ddns", map[string]string{
		"add_ddnsuser_response_failed_ddns_limit_reached.xml":                        "ddns_limit_reached",
		"add_ddnsuser_response_failed_dns_settings_not_allowed.xml":                  "dns_settings_not_allowed",
		"add_ddnsuser_response_failed_dyndns_comment_syntax_incorrect.xml":           "dyndns_comment_syntax_incorrect",
		"add_ddnsuser_response_failed_dyndns_label_not_allowed.xml":                  "dyndns_label_not_allowed",
		"add_ddnsuser_response_failed_dyndns_target_ip_syntax_incorrect.xml":         "dyndns_target_ip_syntax_incorrect",
		"add_ddnsuser_response_failed_missing_parameter.xml":                         "missing_parameter",
		"add_ddnsuser_response_failed_password_syntax_incorrect.xml":                 "password_syntax_incorrect",
		"add_ddnsuser_response_failed_record_name_syntax_incorrect.xml":              "record_name_syntax_incorrect",
		"add_ddnsuser_response_failed_settings_not_in_contract.xml":                  "settings_not_in_contract",
		"update_ddnsuser_response_failed_ddns_service_temporarily_not_available.xml": "ddns_service_temporarily_not_available",
		"update_ddnsuser_response_failed_dns_settings_not_allowed.xml":               "dns_settings_not_allowed",
		"update_ddnsuser_response_failed_dyndns_comment_syntax_incorrect.xml":        "dyndns_comment_syntax_incorrect",
		"update_ddnsuser_response_failed_dyndns_dual_stack_syntax_incorrect.xml":     "dyndns_dual_stack_syntax_incorrect",
		"update_ddnsuser_response_failed_dyndns_login_not_found.xml":                 "dyndns_login_not_found",
		"update_ddnsuser_response_failed_dyndns_target_ip6_syntax_incorrect.xml":     "dyndns_target_ip6_syntax_incorrect",
		"update_ddnsuser_response_failed_dyndns_target_ip_syntax_incorrect.xml":      "dyndns_target_ip_syntax_incorrect",
		"update_ddnsuser_response_failed_missing_parameter.xml":                      "missing_parameter",
		"update_ddnsuser_response_failed_nothing_to_do.xml":                          "nothing_to_do",
		"update_ddnsuser_response_failed_password_syntax_incorrect.xml":              "password_syntax_incorrect",
		"delete_ddnsuser_response_failed_dyndns_login_not_found.xml":                 "dyndns_login_not_found",
		"delete_ddnsuser_response_failed_missing_parameter.xml":                      "missing_parameter",
	})
}
