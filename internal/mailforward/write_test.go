package mailforward_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/mailforward"
	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestClientAdd(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "mailforward/add_mailforward_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	addr, err := mailforward.NewClient(fc).Add(
		context.Background(), "info", "example.de", []string{"to@example.de", "backup@example.de"})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if fc.GotAction != "add_mailforward" {
		t.Errorf("action = %q, want add_mailforward", fc.GotAction)
	}
	if fc.GotParams["local_part"] != "info" || fc.GotParams["domain_part"] != "example.de" {
		t.Errorf("local/domain params = %v", fc.GotParams)
	}
	if fc.GotParams["target_0"] != "to@example.de" || fc.GotParams["target_1"] != "backup@example.de" {
		t.Errorf("target params = %v", fc.GotParams)
	}
	if addr != "info@example.de" {
		t.Errorf("returned address = %q, want info@example.de (fixture ReturnInfo)", addr)
	}
}

func TestClientUpdate(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "mailforward/update_mailforward_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	if err := mailforward.NewClient(fc).Update(
		context.Background(), "info@example.de", []string{"new-target@example.de"}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if fc.GotAction != "update_mailforward" {
		t.Errorf("action = %q, want update_mailforward", fc.GotAction)
	}
	if fc.GotParams["mail_forward"] != "info@example.de" || fc.GotParams["target_0"] != "new-target@example.de" {
		t.Errorf("params = %v", fc.GotParams)
	}
}

func TestClientDelete(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "mailforward/delete_mailforward_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	if err := mailforward.NewClient(fc).Delete(context.Background(), "info@example.de"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if fc.GotAction != "delete_mailforward" {
		t.Errorf("action = %q, want delete_mailforward", fc.GotAction)
	}
	if fc.GotParams["mail_forward"] != "info@example.de" {
		t.Errorf("params = %v", fc.GotParams)
	}
}

func TestWriteValidation(t *testing.T) {
	t.Parallel()
	c := mailforward.NewClient(&testutil.FakeCaller{})
	ctx := context.Background()
	if _, err := c.Add(ctx, "", "example.de", []string{"a@b.de"}); err == nil {
		t.Error("Add empty local part: err = nil, want validation error")
	}
	if _, err := c.Add(ctx, "info", "example.de", nil); err == nil {
		t.Error("Add no targets: err = nil, want validation error")
	}
	if err := c.Update(ctx, "", []string{"a@b.de"}); err == nil {
		t.Error("Update empty address: err = nil, want validation error")
	}
	if err := c.Update(ctx, "info@example.de", nil); err == nil {
		t.Error("Update no targets: err = nil, want validation error")
	}
	if err := c.Delete(ctx, ""); err == nil {
		t.Error("Delete empty address: err = nil, want validation error")
	}
}

func TestUnexpectedReturnString(t *testing.T) {
	t.Parallel()
	resp := &soap.Response{Body: soap.ResponseBody{ReturnString: "FALSE"}}
	c := mailforward.NewClient(&testutil.FakeCaller{Resp: resp})
	ctx := context.Background()
	if _, err := c.Add(ctx, "info", "example.de", []string{"a@b.de"}); !errors.Is(err, mailforward.ErrUnexpectedReturnString) {
		t.Errorf("Add err = %v, want ErrUnexpectedReturnString", err)
	}
	if err := c.Update(ctx, "info@example.de", []string{"a@b.de"}); !errors.Is(err, mailforward.ErrUnexpectedReturnString) {
		t.Errorf("Update err = %v, want ErrUnexpectedReturnString", err)
	}
	if err := c.Delete(ctx, "info@example.de"); !errors.Is(err, mailforward.ErrUnexpectedReturnString) {
		t.Errorf("Delete err = %v, want ErrUnexpectedReturnString", err)
	}
}

func TestWritePropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := mailforward.NewClient(&testutil.FakeCaller{Err: want})
	ctx := context.Background()
	if _, err := c.Add(ctx, "info", "example.de", []string{"a@b.de"}); !errors.Is(err, want) {
		t.Errorf("Add err = %v, want %v", err, want)
	}
	if err := c.Update(ctx, "info@example.de", []string{"a@b.de"}); !errors.Is(err, want) {
		t.Errorf("Update err = %v, want %v", err, want)
	}
	if err := c.Delete(ctx, "info@example.de"); !errors.Is(err, want) {
		t.Errorf("Delete err = %v, want %v", err, want)
	}
}

func TestParamBuilders(t *testing.T) {
	t.Parallel()
	add := mailforward.AddParams("info", "example.de", []string{"a@b.de", "c@d.de"})
	if add["local_part"] != "info" || add["domain_part"] != "example.de" ||
		add["target_0"] != "a@b.de" || add["target_1"] != "c@d.de" {
		t.Errorf("AddParams = %v", add)
	}
	upd := mailforward.UpdateParams("info@example.de", []string{"a@b.de"})
	if upd["mail_forward"] != "info@example.de" || upd["target_0"] != "a@b.de" {
		t.Errorf("UpdateParams = %v", upd)
	}
	del := mailforward.DeleteParams("info@example.de")
	if len(del) != 1 || del["mail_forward"] != "info@example.de" {
		t.Errorf("DeleteParams = %v", del)
	}
}

// TestFaultFixturesDecodeToDocumentedCodes binds the captured
// *_response_failed_*.xml fixtures to the KAS contract: each must
// decode to a *soap.FaultError carrying a non-empty fault code, and a
// representative documented sample must carry the exact code. This is
// the fixture↔contract anchor for the write slice (faults reach the
// domain layer as Caller errors in production).
func TestFaultFixturesDecodeToDocumentedCodes(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(testutil.RepoRoot(t), "testdata", "mailforward")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read fixture dir: %v", err)
	}
	want := map[string]string{
		"add_mailforward_response_failed_missing_parameter.xml":                "missing_parameter",
		"add_mailforward_response_failed_mail_forward_exists_as_forward.xml":   "mail_forward_exists_as_forward",
		"update_mailforward_response_failed_nothing_to_do.xml":                 "nothing_to_do",
		"delete_mailforward_response_failed_mail_forward_not_found_in_kas.xml": "mail_forward_not_found_in_kas",
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
		t.Fatal("no fault fixtures found under testdata/mailforward/")
	}
}
