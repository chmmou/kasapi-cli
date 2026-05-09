package api_test

import (
	"errors"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/api"
	"github.com/chmmou/kasapi-cli/internal/soap"
)

func mkAPIErr(code, detail string) *api.Error {
	fe := &soap.FaultError{Fault: soap.Fault{String: code, Detail: detail}}
	return api.NewErrorForTest("test_action", "https://example.invalid/", fe)
}

func TestPredicateClassesByCode(t *testing.T) {
	cases := []struct {
		code      string
		auth      bool
		flood     bool
		notFound  bool
		syntax    bool
		maxR      bool
		inProg    bool
		missingP  bool
		nothingTD bool
	}{
		{code: "no_auth", auth: true},
		{code: "unknown_session", auth: true},
		{code: "kas_session_invalid", auth: true},
		{code: "kas_access_forbidden", auth: true},
		{code: "got_no_login_data", auth: true},
		{code: "flood_protection", flood: true},
		{code: "in_progress", inProg: true},
		{code: "missing_parameter", missingP: true},
		{code: "nothing_to_do", nothingTD: true},
		{code: "account_login_not_found", notFound: true},
		{code: "kas_login_not_found", notFound: true},
		{code: "domain_doesnt_exist", notFound: true},
		{code: "subdomain_doenst_exist", notFound: true}, // server typo
		{code: "domain_not_found_in_kas", notFound: true},
		{code: "domain_syntax_incorrect", syntax: true},
		{code: "account_kas_password_syntax_incorrect", syntax: true},
		{code: "max_account_reached", maxR: true},
		{code: "max_webspace_reached", maxR: true},
		{code: "ddns_limit_reached", maxR: true},
		{code: "targets_limit_reached", maxR: true},
		// max_*_syntax_incorrect must classify as syntax, not as max-reached.
		{code: "max_database_syntax_incorrect", syntax: true},
	}

	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			e := mkAPIErr(tc.code, "")
			if got := api.IsAuthFailure(e); got != tc.auth {
				t.Errorf("IsAuthFailure = %v, want %v", got, tc.auth)
			}
			if got := api.IsFloodProtection(e); got != tc.flood {
				t.Errorf("IsFloodProtection = %v, want %v", got, tc.flood)
			}
			if got := api.IsNotFound(e); got != tc.notFound {
				t.Errorf("IsNotFound = %v, want %v", got, tc.notFound)
			}
			if got := api.IsSyntaxError(e); got != tc.syntax {
				t.Errorf("IsSyntaxError = %v, want %v", got, tc.syntax)
			}
			if got := api.IsMaxReached(e); got != tc.maxR {
				t.Errorf("IsMaxReached = %v, want %v", got, tc.maxR)
			}
			if got := api.IsInProgress(e); got != tc.inProg {
				t.Errorf("IsInProgress = %v, want %v", got, tc.inProg)
			}
			if got := api.IsMissingParameter(e); got != tc.missingP {
				t.Errorf("IsMissingParameter = %v, want %v", got, tc.missingP)
			}
			if got := api.IsNothingToDo(e); got != tc.nothingTD {
				t.Errorf("IsNothingToDo = %v, want %v", got, tc.nothingTD)
			}
		})
	}
}

func TestPredicatesIgnoreNonAPIErrors(t *testing.T) {
	plain := errors.New("transport: connection reset")
	for name, pred := range map[string]func(error) bool{
		"IsAuthFailure":      api.IsAuthFailure,
		"IsFloodProtection":  api.IsFloodProtection,
		"IsNotFound":         api.IsNotFound,
		"IsSyntaxError":      api.IsSyntaxError,
		"IsMaxReached":       api.IsMaxReached,
		"IsInProgress":       api.IsInProgress,
		"IsMissingParameter": api.IsMissingParameter,
		"IsNothingToDo":      api.IsNothingToDo,
	} {
		if pred(plain) {
			t.Errorf("%s(non-api err) = true, want false", name)
		}
	}
}

func TestErrorMessageIncludesActionAndCode(t *testing.T) {
	e := mkAPIErr("missing_parameter", "kas_action missing")
	if e.Code != "missing_parameter" {
		t.Errorf("Code = %q, want missing_parameter", e.Code)
	}
	if e.Action != "test_action" {
		t.Errorf("Action = %q, want test_action", e.Action)
	}
	msg := e.Error()
	if !errors.Is(e, e) {
		t.Error("errors.Is(e, e) = false")
	}
	for _, want := range []string{"test_action", "missing_parameter", "kas_action missing"} {
		if !contains(msg, want) {
			t.Errorf("Error() = %q, missing %q", msg, want)
		}
	}
	var fe *soap.FaultError
	if !errors.As(e, &fe) {
		t.Error("errors.As to *soap.FaultError failed")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
