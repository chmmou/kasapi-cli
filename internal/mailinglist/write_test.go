package mailinglist_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/mailinglist"
	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestClientAdd(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "mailinglist/add_mailinglist_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	id, err := mailinglist.NewClient(fc).Add(
		context.Background(), "announce", "example.com", "s3cret")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if fc.GotAction != "add_mailinglist" {
		t.Errorf("action = %q, want add_mailinglist", fc.GotAction)
	}
	if fc.GotParams["mailinglist_name"] != "announce" ||
		fc.GotParams["mailinglist_domain"] != "example.com" ||
		fc.GotParams["mailinglist_password"] != "s3cret" {
		t.Errorf("params = %v", fc.GotParams)
	}
	if id != "announce-example-org" {
		t.Errorf("returned id = %q, want announce-example-org (fixture ReturnInfo)", id)
	}
}

func TestClientUpdate(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "mailinglist/update_mailinglist_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	fields := map[string]string{
		mailinglist.FieldSubscriber: "a@example.org\nb@example.org",
		mailinglist.FieldIsActive:   "N",
	}
	if err := mailinglist.NewClient(fc).Update(
		context.Background(), "announce-example-com", fields); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if fc.GotAction != "update_mailinglist" {
		t.Errorf("action = %q, want update_mailinglist", fc.GotAction)
	}
	if fc.GotParams["mailinglist_name"] != "announce-example-com" ||
		fc.GotParams["subscriber"] != "a@example.org\nb@example.org" ||
		fc.GotParams["is_active"] != "N" {
		t.Errorf("params = %v", fc.GotParams)
	}
}

func TestClientDelete(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "mailinglist/delete_mailinglist_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	if err := mailinglist.NewClient(fc).Delete(context.Background(), "announce-example-com"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if fc.GotAction != "delete_mailinglist" {
		t.Errorf("action = %q, want delete_mailinglist", fc.GotAction)
	}
	if fc.GotParams["mailinglist_name"] != "announce-example-com" {
		t.Errorf("params = %v", fc.GotParams)
	}
}

func TestWriteValidation(t *testing.T) {
	t.Parallel()
	c := mailinglist.NewClient(&testutil.FakeCaller{})
	ctx := context.Background()
	if _, err := c.Add(ctx, "", "example.com", "pw"); err == nil {
		t.Error("Add empty name: err = nil, want validation error")
	}
	if _, err := c.Add(ctx, "announce", "", "pw"); err == nil {
		t.Error("Add empty domain: err = nil, want validation error")
	}
	if _, err := c.Add(ctx, "announce", "example.com", ""); err == nil {
		t.Error("Add empty password: err = nil, want validation error")
	}
	if err := c.Update(ctx, "", map[string]string{mailinglist.FieldIsActive: "Y"}); err == nil {
		t.Error("Update empty name: err = nil, want validation error")
	}
	if err := c.Update(ctx, "announce-example-com", nil); err == nil {
		t.Error("Update no fields: err = nil, want validation error")
	}
	if err := c.Delete(ctx, ""); err == nil {
		t.Error("Delete empty name: err = nil, want validation error")
	}
}

func TestUnexpectedReturnString(t *testing.T) {
	t.Parallel()
	resp := &soap.Response{Body: soap.ResponseBody{ReturnString: "FALSE"}}
	c := mailinglist.NewClient(&testutil.FakeCaller{Resp: resp})
	ctx := context.Background()
	if _, err := c.Add(ctx, "announce", "example.com", "pw"); !errors.Is(err, mailinglist.ErrUnexpectedReturnString) {
		t.Errorf("Add err = %v, want ErrUnexpectedReturnString", err)
	}
	if err := c.Update(ctx, "announce-example-com", map[string]string{mailinglist.FieldIsActive: "Y"}); !errors.Is(err, mailinglist.ErrUnexpectedReturnString) {
		t.Errorf("Update err = %v, want ErrUnexpectedReturnString", err)
	}
	if err := c.Delete(ctx, "announce-example-com"); !errors.Is(err, mailinglist.ErrUnexpectedReturnString) {
		t.Errorf("Delete err = %v, want ErrUnexpectedReturnString", err)
	}
}

func TestWritePropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := mailinglist.NewClient(&testutil.FakeCaller{Err: want})
	ctx := context.Background()
	if _, err := c.Add(ctx, "announce", "example.com", "pw"); !errors.Is(err, want) {
		t.Errorf("Add err = %v, want %v", err, want)
	}
	if err := c.Update(ctx, "announce-example-com", map[string]string{mailinglist.FieldIsActive: "Y"}); !errors.Is(err, want) {
		t.Errorf("Update err = %v, want %v", err, want)
	}
	if err := c.Delete(ctx, "announce-example-com"); !errors.Is(err, want) {
		t.Errorf("Delete err = %v, want %v", err, want)
	}
}

func TestParamBuilders(t *testing.T) {
	t.Parallel()
	add := mailinglist.AddParams("announce", "example.com", "pw")
	if add["mailinglist_name"] != "announce" || add["mailinglist_domain"] != "example.com" ||
		add["mailinglist_password"] != "pw" || len(add) != 3 {
		t.Errorf("AddParams = %v", add)
	}
	upd := mailinglist.UpdateParams("announce-example-com", map[string]string{
		mailinglist.FieldSubscriber: "a@b.de",
		mailinglist.FieldIsActive:   "N",
	})
	if upd["mailinglist_name"] != "announce-example-com" ||
		upd["subscriber"] != "a@b.de" || upd["is_active"] != "N" || len(upd) != 3 {
		t.Errorf("UpdateParams = %v", upd)
	}
	del := mailinglist.DeleteParams("announce-example-com")
	if len(del) != 1 || del["mailinglist_name"] != "announce-example-com" {
		t.Errorf("DeleteParams = %v", del)
	}
}

// TestFaultFixturesDecodeToDocumentedCodes binds the captured
// *_response_failed_*.xml fixtures to the KAS contract: each must
// decode to a *soap.FaultError carrying a non-empty fault code, and a
// representative documented sample must carry the exact code. The
// add_mailinglist_..._mailinglist_mailinglist_domain_doesnt_exist
// fixture is the reason this is an explicit map rather than a
// filename-suffix derivation: its filename duplicates the mailinglist_
// prefix while the fault code does not.
func TestFaultFixturesDecodeToDocumentedCodes(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(testutil.RepoRoot(t), "testdata", "mailinglist")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read fixture dir: %v", err)
	}
	want := map[string]string{
		"add_mailinglist_response_failed_missing_parameter.xml":                           "missing_parameter",
		"add_mailinglist_response_failed_mailinglist_mailinglist_domain_doesnt_exist.xml": "mailinglist_domain_doesnt_exist",
		"update_mailinglist_response_failed_nothing_to_do.xml":                            "nothing_to_do",
		"update_mailinglist_response_failed_subscriber_email_syntax_incorrect.xml":        "subscriber_email_syntax_incorrect",
		"delete_mailinglist_response_failed_mailinglist_not_found.xml":                    "mailinglist_not_found",
	}
	seen := 0
	for _, e := range entries {
		name := e.Name()
		if !strings.Contains(name, "_response_failed_") {
			continue
		}
		seen++
		//nolint:gosec // G304: fixture path is rooted at the repo testdata/ dir.
		f, oerr := os.Open(filepath.Join(dir, name))
		if oerr != nil {
			t.Fatalf("open %s: %v", name, oerr)
		}
		_, derr := soap.Decode(f)
		_ = f.Close()
		var fe *soap.FaultError
		if !errors.As(derr, &fe) {
			t.Errorf("%s: decode err = %v, want *soap.FaultError", name, derr)
			continue
		}
		if fe.Fault.String == "" {
			t.Errorf("%s: empty fault code", name)
		}
		if code, ok := want[name]; ok && fe.Fault.String != code {
			t.Errorf("%s: fault = %q, want %q", name, fe.Fault.String, code)
		}
	}
	if seen == 0 {
		t.Fatal("no fault fixtures found under testdata/mailinglist/")
	}
}
