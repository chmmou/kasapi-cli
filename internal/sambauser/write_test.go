package sambauser_test

import (
	"context"
	"errors"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/sambauser"
	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func sampleSpec() sambauser.Spec {
	return sambauser.Spec{
		Password: "s3cret",
		Comment:  "Shared documents",
		Path:     "/example.com/share/",
	}
}

func TestClientAdd(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "sambauser/add_sambauser_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	login, err := sambauser.NewClient(fc).Add(context.Background(), sampleSpec())
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if fc.GotAction != "add_sambauser" {
		t.Errorf("action = %q, want add_sambauser", fc.GotAction)
	}
	if fc.GotParams["samba_password"] != "s3cret" || fc.GotParams["samba_comment"] != "Shared documents" ||
		fc.GotParams["samba_path"] != "/example.com/share/" {
		t.Errorf("params = %v", fc.GotParams)
	}
	// The KAS docs wrongly name the add password samba_new_password;
	// the real key (per the corrected fixture) is samba_password.
	if _, ok := fc.GotParams["samba_new_password"]; ok {
		t.Errorf("add_sambauser must use samba_password, not samba_new_password: %v", fc.GotParams)
	}
	if _, ok := fc.GotParams["samba_login"]; ok {
		t.Errorf("add_sambauser must not send samba_login (server generates it): %v", fc.GotParams)
	}
	if login != "s0000003" {
		t.Errorf("returned login = %q, want s0000003 (fixture ReturnInfo)", login)
	}
}

func TestClientUpdate(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "sambauser/update_sambauser_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	fields := map[string]string{
		sambauser.FieldComment:     "Renamed shared documents",
		sambauser.FieldNewPassword: "n3wpass",
	}
	if err := sambauser.NewClient(fc).Update(context.Background(), "s0000000", fields); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if fc.GotAction != "update_sambauser" {
		t.Errorf("action = %q, want update_sambauser", fc.GotAction)
	}
	if fc.GotParams["samba_login"] != "s0000000" ||
		fc.GotParams["samba_comment"] != "Renamed shared documents" ||
		fc.GotParams["samba_new_password"] != "n3wpass" {
		t.Errorf("params = %v", fc.GotParams)
	}
	if _, ok := fc.GotParams["samba_password"]; ok {
		t.Errorf("update_sambauser must use samba_new_password, not samba_password: %v", fc.GotParams)
	}
}

func TestClientDelete(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "sambauser/delete_sambauser_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	if err := sambauser.NewClient(fc).Delete(context.Background(), "s0000000"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if fc.GotAction != "delete_sambauser" {
		t.Errorf("action = %q, want delete_sambauser", fc.GotAction)
	}
	if fc.GotParams["samba_login"] != "s0000000" {
		t.Errorf("params = %v", fc.GotParams)
	}
}

func TestWriteValidation(t *testing.T) {
	t.Parallel()
	c := sambauser.NewClient(&testutil.FakeCaller{})
	ctx := context.Background()

	for _, tc := range []struct {
		name string
		mut  func(*sambauser.Spec)
	}{
		{"missing password", func(s *sambauser.Spec) { s.Password = "" }},
		{"missing comment", func(s *sambauser.Spec) { s.Comment = "" }},
		{"missing path", func(s *sambauser.Spec) { s.Path = "" }},
	} {
		s := sampleSpec()
		tc.mut(&s)
		if _, err := c.Add(ctx, s); err == nil {
			t.Errorf("Add %s: err = nil, want validation error", tc.name)
		}
	}
	if err := c.Update(ctx, "", map[string]string{sambauser.FieldComment: "x"}); err == nil {
		t.Error("Update empty login: err = nil, want validation error")
	}
	if err := c.Update(ctx, "s0000000", nil); err == nil {
		t.Error("Update no fields: err = nil, want validation error")
	}
	if err := c.Delete(ctx, ""); err == nil {
		t.Error("Delete empty login: err = nil, want validation error")
	}
}

func TestUnexpectedReturnString(t *testing.T) {
	t.Parallel()
	resp := &soap.Response{Body: soap.ResponseBody{ReturnString: "FALSE"}}
	c := sambauser.NewClient(&testutil.FakeCaller{Resp: resp})
	ctx := context.Background()
	if _, err := c.Add(ctx, sampleSpec()); !errors.Is(err, sambauser.ErrUnexpectedReturnString) {
		t.Errorf("Add err = %v, want ErrUnexpectedReturnString", err)
	}
	if err := c.Update(ctx, "s0000000", map[string]string{sambauser.FieldComment: "x"}); !errors.Is(err, sambauser.ErrUnexpectedReturnString) {
		t.Errorf("Update err = %v, want ErrUnexpectedReturnString", err)
	}
	if err := c.Delete(ctx, "s0000000"); !errors.Is(err, sambauser.ErrUnexpectedReturnString) {
		t.Errorf("Delete err = %v, want ErrUnexpectedReturnString", err)
	}
}

func TestWritePropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := sambauser.NewClient(&testutil.FakeCaller{Err: want})
	ctx := context.Background()
	if _, err := c.Add(ctx, sampleSpec()); !errors.Is(err, want) {
		t.Errorf("Add err = %v, want %v", err, want)
	}
	if err := c.Update(ctx, "s0000000", map[string]string{sambauser.FieldComment: "x"}); !errors.Is(err, want) {
		t.Errorf("Update err = %v, want %v", err, want)
	}
	if err := c.Delete(ctx, "s0000000"); !errors.Is(err, want) {
		t.Errorf("Delete err = %v, want %v", err, want)
	}
}

func TestParamBuilders(t *testing.T) {
	t.Parallel()
	add := sambauser.AddParams(sampleSpec())
	if add["samba_password"] != "s3cret" || add["samba_comment"] != "Shared documents" ||
		add["samba_path"] != "/example.com/share/" {
		t.Errorf("AddParams = %v", add)
	}
	if len(add) != 3 {
		t.Errorf("AddParams has %d keys, want 3 (samba_password/comment/path, no samba_login)", len(add))
	}
	if _, ok := add["samba_login"]; ok {
		t.Errorf("AddParams must not contain samba_login: %v", add)
	}
	upd := sambauser.UpdateParams("s0000000", map[string]string{
		sambauser.FieldComment: "c", sambauser.FieldNewPassword: "p",
	})
	if upd["samba_login"] != "s0000000" || upd["samba_comment"] != "c" || upd["samba_new_password"] != "p" {
		t.Errorf("UpdateParams = %v", upd)
	}
	del := sambauser.DeleteParams("s0000000")
	if len(del) != 1 || del["samba_login"] != "s0000000" {
		t.Errorf("DeleteParams = %v", del)
	}
}
