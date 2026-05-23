package mailfilter_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/mailfilter"
	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func sampleFilters() []string {
	return []string{
		"pdw",
		"virus_mark",
		"spamc_move:move=Spam",
		"rspamd:move=Spam",
		"brbl:move=Spam",
		"rbl_cbl:move=Spam",
		"scbl:move=Spam",
	}
}

func TestClientAdd(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "mailfilter/add_mailstandardfilter_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	err := mailfilter.NewClient(fc).Add(context.Background(), "m0000001", sampleFilters())
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if fc.GotAction != "add_mailstandardfilter" {
		t.Errorf("action = %q, want add_mailstandardfilter", fc.GotAction)
	}
	if got, _ := fc.GotParams["mail_login"].(string); got != "m0000001" {
		t.Errorf("params[mail_login] = %v, want m0000001", fc.GotParams["mail_login"])
	}
	wantFilter := "pdw;virus_mark;spamc_move:move=Spam;rspamd:move=Spam;brbl:move=Spam;rbl_cbl:move=Spam;scbl:move=Spam"
	if got, _ := fc.GotParams["filter"].(string); got != wantFilter {
		t.Errorf("params[filter] = %q, want %q", fc.GotParams["filter"], wantFilter)
	}
}

func TestClientDelete(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "mailfilter/delete_mailstandardfilter_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	err := mailfilter.NewClient(fc).Delete(context.Background(), "m0000001")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if fc.GotAction != "delete_mailstandardfilter" {
		t.Errorf("action = %q, want delete_mailstandardfilter", fc.GotAction)
	}
	if got, _ := fc.GotParams["mail_login"].(string); got != "m0000001" {
		t.Errorf("params[mail_login] = %v, want m0000001", fc.GotParams["mail_login"])
	}
	if _, ok := fc.GotParams["filter"]; ok {
		t.Errorf("delete params must not carry a filter key, got %v", fc.GotParams)
	}
}

// TestClientDeleteSurfacesInternalServerFault locks in the documented
// quirk: delete_mailstandardfilter sometimes returns a generic
// envelope-level SOAP fault even though the chain was in fact removed on
// the server. We decode the captured shared fixture
// (testdata/response_failed_internal_server_error.xml) the same way the
// real *api.Client does (soap.Decode → *soap.FaultError) and route that
// error through the FakeCaller, then assert Delete surfaces it
// verbatim. A future client-side "treat as success" shortcut would
// have to remove this assertion deliberately.
func TestClientDeleteSurfacesInternalServerFault(t *testing.T) {
	t.Parallel()
	path := filepath.Join(testutil.RepoRoot(t), "testdata", "response_failed_internal_server_error.xml")
	//nolint:gosec // G304: rooted at the repo testdata/ dir.
	f, oerr := os.Open(path)
	if oerr != nil {
		t.Fatalf("open fixture: %v", oerr)
	}
	defer func() { _ = f.Close() }()
	_, derr := soap.Decode(f)
	var fe *soap.FaultError
	if !errors.As(derr, &fe) {
		t.Fatalf("soap.Decode err = %T (%v), want *soap.FaultError", derr, derr)
	}
	if !strings.Contains(fe.Fault.String, "sizeof()") {
		t.Fatalf("fixture FaultError.String = %q, want the captured PHP message", fe.Fault.String)
	}
	fc := &testutil.FakeCaller{Err: fe}
	err := mailfilter.NewClient(fc).Delete(context.Background(), "m0000001")
	if err == nil {
		t.Fatal("Delete err = nil, want the internal-server-error fault surfaced")
	}
	var got *soap.FaultError
	if !errors.As(err, &got) {
		t.Fatalf("Delete err = %T (%v), want *soap.FaultError wrapped", err, err)
	}
	if !strings.Contains(got.Fault.String, "sizeof()") {
		t.Errorf("propagated FaultError.String = %q, want the captured PHP message", got.Fault.String)
	}
}

func TestWriteValidation(t *testing.T) {
	t.Parallel()
	c := mailfilter.NewClient(&testutil.FakeCaller{})
	ctx := context.Background()

	if err := c.Add(ctx, "", sampleFilters()); err == nil ||
		!strings.Contains(err.Error(), "mail login") {
		t.Errorf("Add empty login err = %v, want mail-login validation", err)
	}
	if err := c.Add(ctx, "m0000001", nil); err == nil ||
		!strings.Contains(err.Error(), "filter item") {
		t.Errorf("Add no filters err = %v, want filter-item validation", err)
	}
	if err := c.Add(ctx, "m0000001", []string{"pdw", ""}); err == nil ||
		!strings.Contains(err.Error(), "empty") {
		t.Errorf("Add empty item err = %v, want empty-item validation", err)
	}
	if err := c.Add(ctx, "m0000001", []string{"pdw;virus_mark"}); err == nil ||
		!strings.Contains(err.Error(), ";") {
		t.Errorf("Add semicolon-in-item err = %v, want semicolon validation", err)
	}
	if err := c.Delete(ctx, ""); err == nil ||
		!strings.Contains(err.Error(), "mail login") {
		t.Errorf("Delete empty login err = %v, want mail-login validation", err)
	}
}

func TestUnexpectedReturnString(t *testing.T) {
	t.Parallel()
	// Synthetic response with ReturnString != "TRUE" — kaswrite must
	// wrap this into ErrUnexpectedReturnString.
	bad := &soap.Response{Body: soap.ResponseBody{ReturnString: "FALSE"}}
	for _, tc := range []struct {
		name string
		run  func(*mailfilter.Client) error
	}{
		{"Add", func(c *mailfilter.Client) error {
			return c.Add(context.Background(), "m0000001", sampleFilters())
		}},
		{"Delete", func(c *mailfilter.Client) error {
			return c.Delete(context.Background(), "m0000001")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := mailfilter.NewClient(&testutil.FakeCaller{Resp: bad})
			if err := tc.run(c); !errors.Is(err, mailfilter.ErrUnexpectedReturnString) {
				t.Errorf("%s err = %v, want errors.Is ErrUnexpectedReturnString", tc.name, err)
			}
		})
	}
}

func TestWritePropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := mailfilter.NewClient(&testutil.FakeCaller{Err: want})
	if err := c.Add(context.Background(), "m0000001", sampleFilters()); !errors.Is(err, want) {
		t.Errorf("Add err = %v, want %v wrapped", err, want)
	}
	if err := c.Delete(context.Background(), "m0000001"); !errors.Is(err, want) {
		t.Errorf("Delete err = %v, want %v wrapped", err, want)
	}
}

func TestParamBuilders(t *testing.T) {
	t.Parallel()
	addParams := mailfilter.AddParams("m0000001", "pdw;virus_mark")
	if got, _ := addParams["mail_login"].(string); got != "m0000001" {
		t.Errorf("AddParams[mail_login] = %v, want m0000001", addParams["mail_login"])
	}
	if got, _ := addParams["filter"].(string); got != "pdw;virus_mark" {
		t.Errorf("AddParams[filter] = %v, want pdw;virus_mark", addParams["filter"])
	}
	delParams := mailfilter.DeleteParams("m0000001")
	if got, _ := delParams["mail_login"].(string); got != "m0000001" {
		t.Errorf("DeleteParams[mail_login] = %v, want m0000001", delParams["mail_login"])
	}
	if _, ok := delParams["filter"]; ok {
		t.Errorf("DeleteParams must not carry a filter key, got %v", delParams)
	}
}

func TestJoinFilters(t *testing.T) {
	t.Parallel()
	got, err := mailfilter.JoinFilters([]string{"a", "b:opt=x", "c"})
	if err != nil {
		t.Fatalf("JoinFilters: %v", err)
	}
	if got != "a;b:opt=x;c" {
		t.Errorf("JoinFilters = %q, want %q", got, "a;b:opt=x;c")
	}
	if _, err := mailfilter.JoinFilters(nil); err == nil {
		t.Errorf("JoinFilters(nil) err = nil, want validation error")
	}
}
