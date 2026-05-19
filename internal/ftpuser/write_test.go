package ftpuser_test

import (
	"context"
	"errors"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/ftpuser"
	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func sampleSpec() ftpuser.Spec {
	return ftpuser.Spec{
		Password:        "s3cret",
		Comment:         "FTP User Test Only",
		Path:            "/ftp/path/",
		PermissionRead:  "Y",
		PermissionWrite: "Y",
		PermissionList:  "Y",
		VirusClamAV:     "Y",
	}
}

func TestClientAdd(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "ftpuser/add_ftpuser_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	login, err := ftpuser.NewClient(fc).Add(context.Background(), sampleSpec())
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if fc.GotAction != "add_ftpuser" {
		t.Errorf("action = %q, want add_ftpuser (singular — live API confirmed)", fc.GotAction)
	}
	if fc.GotParams["ftp_password"] != "s3cret" || fc.GotParams["ftp_comment"] != "FTP User Test Only" ||
		fc.GotParams["ftp_path"] != "/ftp/path/" || fc.GotParams["ftp_virus_clamav"] != "Y" {
		t.Errorf("params = %v", fc.GotParams)
	}
	if _, ok := fc.GotParams["ftp_login"]; ok {
		t.Errorf("add_ftpuser must not send ftp_login (server generates it): %v", fc.GotParams)
	}
	if login != "f0000004" {
		t.Errorf("returned login = %q, want f0000004 (fixture ReturnInfo)", login)
	}
}

func TestClientUpdate(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "ftpuser/update_ftpuser_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	fields := map[string]string{
		ftpuser.FieldComment:     "FTP User Write Only Update",
		ftpuser.FieldNewPassword: "n3wpass",
	}
	if err := ftpuser.NewClient(fc).Update(context.Background(), "f0000004", fields); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if fc.GotAction != "update_ftpuser" {
		t.Errorf("action = %q, want update_ftpuser", fc.GotAction)
	}
	if fc.GotParams["ftp_login"] != "f0000004" ||
		fc.GotParams["ftp_comment"] != "FTP User Write Only Update" ||
		fc.GotParams["ftp_new_password"] != "n3wpass" {
		t.Errorf("params = %v", fc.GotParams)
	}
	if _, ok := fc.GotParams["ftp_password"]; ok {
		t.Errorf("update_ftpuser must use ftp_new_password, not ftp_password: %v", fc.GotParams)
	}
}

func TestClientDelete(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "ftpuser/delete_ftpuser_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	if err := ftpuser.NewClient(fc).Delete(context.Background(), "f0000004"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if fc.GotAction != "delete_ftpuser" {
		t.Errorf("action = %q, want delete_ftpuser", fc.GotAction)
	}
	if fc.GotParams["ftp_login"] != "f0000004" {
		t.Errorf("params = %v", fc.GotParams)
	}
}

func TestWriteValidation(t *testing.T) {
	t.Parallel()
	c := ftpuser.NewClient(&testutil.FakeCaller{})
	ctx := context.Background()

	missingPassword := sampleSpec()
	missingPassword.Password = ""
	if _, err := c.Add(ctx, missingPassword); err == nil {
		t.Error("Add missing password: err = nil, want validation error")
	}
	missingComment := sampleSpec()
	missingComment.Comment = ""
	if _, err := c.Add(ctx, missingComment); err == nil {
		t.Error("Add missing comment: err = nil, want validation error")
	}
	if err := c.Update(ctx, "", map[string]string{ftpuser.FieldComment: "x"}); err == nil {
		t.Error("Update empty login: err = nil, want validation error")
	}
	if err := c.Update(ctx, "f0000004", nil); err == nil {
		t.Error("Update no fields: err = nil, want validation error")
	}
	if err := c.Delete(ctx, ""); err == nil {
		t.Error("Delete empty login: err = nil, want validation error")
	}
}

func TestUnexpectedReturnString(t *testing.T) {
	t.Parallel()
	resp := &soap.Response{Body: soap.ResponseBody{ReturnString: "FALSE"}}
	c := ftpuser.NewClient(&testutil.FakeCaller{Resp: resp})
	ctx := context.Background()
	if _, err := c.Add(ctx, sampleSpec()); !errors.Is(err, ftpuser.ErrUnexpectedReturnString) {
		t.Errorf("Add err = %v, want ErrUnexpectedReturnString", err)
	}
	if err := c.Update(ctx, "f0000004", map[string]string{ftpuser.FieldComment: "x"}); !errors.Is(err, ftpuser.ErrUnexpectedReturnString) {
		t.Errorf("Update err = %v, want ErrUnexpectedReturnString", err)
	}
	if err := c.Delete(ctx, "f0000004"); !errors.Is(err, ftpuser.ErrUnexpectedReturnString) {
		t.Errorf("Delete err = %v, want ErrUnexpectedReturnString", err)
	}
}

func TestWritePropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := ftpuser.NewClient(&testutil.FakeCaller{Err: want})
	ctx := context.Background()
	if _, err := c.Add(ctx, sampleSpec()); !errors.Is(err, want) {
		t.Errorf("Add err = %v, want %v", err, want)
	}
	if err := c.Update(ctx, "f0000004", map[string]string{ftpuser.FieldComment: "x"}); !errors.Is(err, want) {
		t.Errorf("Update err = %v, want %v", err, want)
	}
	if err := c.Delete(ctx, "f0000004"); !errors.Is(err, want) {
		t.Errorf("Delete err = %v, want %v", err, want)
	}
}

func TestParamBuilders(t *testing.T) {
	t.Parallel()
	add := ftpuser.AddParams(sampleSpec())
	if add["ftp_password"] != "s3cret" || add["ftp_comment"] != "FTP User Test Only" ||
		add["ftp_path"] != "/ftp/path/" || add["ftp_permission_read"] != "Y" {
		t.Errorf("AddParams = %v", add)
	}
	if len(add) != 7 {
		t.Errorf("AddParams has %d keys, want 7 (all add_ftpuser request fields, no ftp_login)", len(add))
	}
	if _, ok := add["ftp_login"]; ok {
		t.Errorf("AddParams must not contain ftp_login: %v", add)
	}
	upd := ftpuser.UpdateParams("f0000004", map[string]string{
		ftpuser.FieldComment: "c", ftpuser.FieldNewPassword: "p",
	})
	if upd["ftp_login"] != "f0000004" || upd["ftp_comment"] != "c" || upd["ftp_new_password"] != "p" {
		t.Errorf("UpdateParams = %v", upd)
	}
	del := ftpuser.DeleteParams("f0000004")
	if len(del) != 1 || del["ftp_login"] != "f0000004" {
		t.Errorf("DeleteParams = %v", del)
	}
}
