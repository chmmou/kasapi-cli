package mailaccount_test

import (
	"testing"

	"github.com/chmmou/kasapi-cli/internal/testutil"
)

// TestFaultFixturesDecodeToDocumentedCodes pins every captured
// mailaccount fault fixture to the documented KAS fault code it carries.
// AssertFaultFixtures decodes each _response_failed_ file in the module
// dir to a *soap.FaultError (and checks the code is non-empty); listing
// every file here additionally asserts the exact code, so the fixtures
// stay the authoritative contract for the get/add/update/delete actions.
func TestFaultFixturesDecodeToDocumentedCodes(t *testing.T) {
	t.Parallel()
	//nolint:gosec // G101: these map values are KAS fault-code strings (e.g. "webmail_autologin_change_requires_new_password"), not hardcoded credentials.
	testutil.AssertFaultFixtures(t, "mailaccount", map[string]string{
		"add_mailaccount_response_failed_copy_adress_like_mailaccount.xml":                      "copy_adress_like_mailaccount",
		"add_mailaccount_response_failed_copy_adress_syntax_incorrect.xml":                      "copy_adress_syntax_incorrect",
		"add_mailaccount_response_failed_couldnt_get_kas_ressources.xml":                        "couldnt_get_kas_ressources",
		"add_mailaccount_response_failed_email_already_exists.xml":                              "email_already_exists",
		"add_mailaccount_response_failed_email_domain_doesnt_exist.xml":                         "email_domain_doesnt_exist",
		"add_mailaccount_response_failed_email_syntax_incorrect.xml":                            "email_syntax_incorrect",
		"add_mailaccount_response_failed_mail_loop_detected.xml":                                "mail_loop_detected",
		"add_mailaccount_response_failed_mail_xlist_archiv_syntax_incorrect.xml":                "mail_xlist_archiv_syntax_incorrect",
		"add_mailaccount_response_failed_mail_xlist_drafts_syntax_incorrect.xml":                "mail_xlist_drafts_syntax_incorrect",
		"add_mailaccount_response_failed_mail_xlist_duplicate_folder.xml":                       "mail_xlist_duplicate_folder",
		"add_mailaccount_response_failed_mail_xlist_enabled_syntax_incorrect.xml":               "mail_xlist_enabled_syntax_incorrect",
		"add_mailaccount_response_failed_mail_xlist_sent_syntax_incorrect.xml":                  "mail_xlist_sent_syntax_incorrect",
		"add_mailaccount_response_failed_mail_xlist_spam_syntax_incorrect.xml":                  "mail_xlist_spam_syntax_incorrect",
		"add_mailaccount_response_failed_mail_xlist_trash_syntax_incorrect.xml":                 "mail_xlist_trash_syntax_incorrect",
		"add_mailaccount_response_failed_max_emails_reached.xml":                                "max_emails_reached",
		"add_mailaccount_response_failed_max_sender_alias_reached.xml":                          "max_sender_alias_reached",
		"add_mailaccount_response_failed_missing_parameter.xml":                                 "missing_parameter",
		"add_mailaccount_response_failed_password_syntax_incorrect.xml":                         "password_syntax_incorrect",
		"add_mailaccount_response_failed_responder_contentype_syntax_incorrect.xml":             "responder_contentype_syntax_incorrect",
		"add_mailaccount_response_failed_responder_displayname_syntax_incorrect.xml":            "responder_displayname_syntax_incorrect",
		"add_mailaccount_response_failed_responder_not_allowed_for_catchall_adresses.xml":       "responder_not_allowed_for_catchall_adresses",
		"add_mailaccount_response_failed_responder_startdate_gt_enddate.xml":                    "responder_startdate_gt_enddate",
		"add_mailaccount_response_failed_responder_syntax_incorrect.xml":                        "responder_syntax_incorrect",
		"add_mailaccount_response_failed_responder_text_is_empty.xml":                           "responder_text_is_empty",
		"add_mailaccount_response_failed_sender_alias_domain_in_kas.xml":                        "sender_alias_domain_in_kas",
		"add_mailaccount_response_failed_sender_alias_syntax_incorrect.xml":                     "sender_alias_syntax_incorrect",
		"add_mailaccount_response_failed_webmail_autologin_syntax_incorrect.xml":                "webmail_autologin_syntax_incorrect",
		"delete_mailaccount_response_failed_in_progress.xml":                                    "in_progress",
		"delete_mailaccount_response_failed_mail_login_not_found.xml":                           "mail_login_not_found",
		"delete_mailaccount_response_failed_mail_loop_detected.xml":                             "mail_loop_detected",
		"delete_mailaccount_response_failed_missing_parameter.xml":                              "missing_parameter",
		"update_mailaccount_response_failed_copy_adress_like_mailaccount.xml":                   "copy_adress_like_mailaccount",
		"update_mailaccount_response_failed_copy_adress_syntax_incorrect.xml":                   "copy_adress_syntax_incorrect",
		"update_mailaccount_response_failed_email_already_exists.xml":                           "email_already_exists",
		"update_mailaccount_response_failed_email_domain_doesnt_exist.xml":                      "email_domain_doesnt_exist",
		"update_mailaccount_response_failed_in_progress.xml":                                    "in_progress",
		"update_mailaccount_response_failed_is_active_syntax_incorrect.xml":                     "is_active_syntax_incorrect",
		"update_mailaccount_response_failed_mail_login_not_found.xml":                           "mail_login_not_found",
		"update_mailaccount_response_failed_mail_loop_detected.xml":                             "mail_loop_detected",
		"update_mailaccount_response_failed_mail_xlist_archiv_syntax_incorrect.xml":             "mail_xlist_archiv_syntax_incorrect",
		"update_mailaccount_response_failed_mail_xlist_drafts_syntax_incorrect.xml":             "mail_xlist_drafts_syntax_incorrect",
		"update_mailaccount_response_failed_mail_xlist_duplicate_folder.xml":                    "mail_xlist_duplicate_folder",
		"update_mailaccount_response_failed_mail_xlist_enabled_syntax_incorrect.xml":            "mail_xlist_enabled_syntax_incorrect",
		"update_mailaccount_response_failed_mail_xlist_sent_syntax_incorrect.xml":               "mail_xlist_sent_syntax_incorrect",
		"update_mailaccount_response_failed_mail_xlist_spam_syntax_incorrect.xml":               "mail_xlist_spam_syntax_incorrect",
		"update_mailaccount_response_failed_mail_xlist_trash_syntax_incorrect.xml":              "mail_xlist_trash_syntax_incorrect",
		"update_mailaccount_response_failed_max_sender_alias_reached.xml":                       "max_sender_alias_reached",
		"update_mailaccount_response_failed_missing_parameter.xml":                              "missing_parameter",
		"update_mailaccount_response_failed_nothing_to_do.xml":                                  "nothing_to_do",
		"update_mailaccount_response_failed_password_syntax_incorrect.xml":                      "password_syntax_incorrect",
		"update_mailaccount_response_failed_responder_contentype_syntax_incorrect.xml":          "responder_contentype_syntax_incorrect",
		"update_mailaccount_response_failed_responder_displayname_syntax_incorrect.xml":         "responder_displayname_syntax_incorrect",
		"update_mailaccount_response_failed_responder_not_allowed_for_catchall_adresses.xml":    "responder_not_allowed_for_catchall_adresses",
		"update_mailaccount_response_failed_responder_startdate_gt_enddate.xml":                 "responder_startdate_gt_enddate",
		"update_mailaccount_response_failed_responder_syntax_incorrect.xml":                     "responder_syntax_incorrect",
		"update_mailaccount_response_failed_responder_text_is_empty.xml":                        "responder_text_is_empty",
		"update_mailaccount_response_failed_sender_alias_domain_in_kas.xml":                     "sender_alias_domain_in_kas",
		"update_mailaccount_response_failed_sender_alias_syntax_incorrect.xml":                  "sender_alias_syntax_incorrect",
		"update_mailaccount_response_failed_webmail_autologin_change_requires_new_password.xml": "webmail_autologin_change_requires_new_password",
		"update_mailaccount_response_failed_webmail_autologin_syntax_incorrect.xml":             "webmail_autologin_syntax_incorrect",
	})
}
