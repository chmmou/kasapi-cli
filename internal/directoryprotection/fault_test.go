package directoryprotection_test

import (
	"testing"

	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestFaultFixturesDecodeToDocumentedCodes(t *testing.T) {
	t.Parallel()
	testutil.AssertFaultFixtures(t, "directoryprotection", map[string]string{
		// add — including the faults distinctive to the multi-user
		// array form KAS accepts (count mismatch, duplicate, max
		// reached) even though this slice only sends the scalar form.
		"add_directoryprotection_response_failed_directory_authname_syntax_incorrect.xml": "directory_authname_syntax_incorrect",
		"add_directoryprotection_response_failed_directory_password_syntax_incorrect.xml": "directory_password_syntax_incorrect",
		"add_directoryprotection_response_failed_directory_path_syntax_incorrect.xml":     "directory_path_syntax_incorrect",
		"add_directoryprotection_response_failed_directory_user_syntax_incorrect.xml":     "directory_user_syntax_incorrect",
		"add_directoryprotection_response_failed_directory_user_count_neq_passcount.xml":  "directory_user_count_neq_passcount",
		"add_directoryprotection_response_failed_duplicate_directory_user.xml":            "duplicate_directory_user",
		"add_directoryprotection_response_failed_max_directory_user_reached.xml":          "max_directory_user_reached",
		"add_directoryprotection_response_failed_missing_parameter.xml":                   "missing_parameter",
		// update — nothing_to_do is the sparse-update fault (the API
		// rejects an update that changes nothing).
		"update_directoryprotection_response_failed_nothing_to_do.xml":                       "nothing_to_do",
		"update_directoryprotection_response_failed_in_progress.xml":                         "in_progress",
		"update_directoryprotection_response_failed_directory_password_syntax_incorrect.xml": "directory_password_syntax_incorrect",
		// delete
		"delete_directoryprotection_response_failed_in_progress.xml":       "in_progress",
		"delete_directoryprotection_response_failed_missing_parameter.xml": "missing_parameter",
	})
}
