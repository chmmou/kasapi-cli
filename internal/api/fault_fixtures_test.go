package api_test

import (
	"testing"

	"github.com/chmmou/kasapi-cli/internal/api"
	"github.com/chmmou/kasapi-cli/internal/testutil"
)

// TestSharedFaultFixtures anchors the shared top-level testdata/
// response_failed_*.xml set (the cross-module session/auth/action
// faults) to the KAS contract, pinning each fixture to the api.Code*
// constant the error helpers classify it by. Module-specific fault
// fixtures are anchored by the per-module fault tests instead.
func TestSharedFaultFixtures(t *testing.T) {
	t.Parallel()
	testutil.AssertFaultFixtures(t, "", map[string]string{
		"response_failed_no_auth.xml":              api.CodeNoAuth,
		"response_failed_kas_session_invalid.xml":  api.CodeSessionInvalid,
		"response_failed_got_no_login_data.xml":    api.CodeGotNoLoginData,
		"response_failed_kas_access_forbidden.xml": api.CodeAccessForbidden,
		"response_failed_no_action.xml":            api.CodeNoAction,
		"response_failed_unkown_action.xml":        api.CodeUnknownAction,
	})
}
