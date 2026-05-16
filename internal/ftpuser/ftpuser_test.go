package ftpuser_test

import (
	"context"
	"errors"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/ftpuser"
	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestDecodeFTPUsers(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "ftpuser/get_ftpusers_response_success.xml")
	got, err := ftpuser.DecodeFTPUsers(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeFTPUsers: %v", err)
	}
	if len(got) != 5 {
		t.Fatalf("len = %d, want 5", len(got))
	}
	main := got[0]
	if main.Login != "w0000000" {
		t.Errorf("Login = %q, want w0000000", main.Login)
	}
	if main.IsMainUser != "Y" {
		t.Errorf("IsMainUser = %q, want Y", main.IsMainUser)
	}
	listOnly := got[1]
	if listOnly.Login != "f0000001" {
		t.Errorf("Login = %q, want f0000001", listOnly.Login)
	}
	if listOnly.Comment != "FTP User List Only" {
		t.Errorf("Comment = %q", listOnly.Comment)
	}
	if listOnly.PermissionList != "Y" || listOnly.PermissionRead != "N" || listOnly.PermissionWrite != "N" {
		t.Errorf("permissions = L=%s R=%s W=%s, want L=Y R=N W=N",
			listOnly.PermissionList, listOnly.PermissionRead, listOnly.PermissionWrite)
	}
}

func TestDecodeFTPUserSingular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "ftpuser/get_ftpuser_response_success.xml")
	got, err := ftpuser.DecodeFTPUsers(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeFTPUsers: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	u := got[0]
	if u.Login != "f0000001" {
		t.Errorf("Login = %q, want f0000001", u.Login)
	}
	if u.Path != "/ftp/path/" {
		t.Errorf("Path = %q", u.Path)
	}
	if u.VirusClamAV != "Y" {
		t.Errorf("VirusClamAV = %q, want Y", u.VirusClamAV)
	}
	// The legacy ftp_passwort key must be preserved alongside ftp_password.
	if u.Password == "" || u.Passwort == "" {
		t.Errorf("expected both ftp_password and ftp_passwort populated, got %q / %q",
			u.Password, u.Passwort)
	}
}

func TestDecodeFTPUsersEmptyList(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "ftpuser/get_ftpuser_response_success_empty_list.xml")
	got, err := ftpuser.DecodeFTPUsers(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeFTPUsers: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}

func TestClientList(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "ftpuser/get_ftpusers_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	list, err := ftpuser.NewClient(fc).List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if fc.GotAction != "get_ftpusers" {
		t.Errorf("action = %q, want get_ftpusers", fc.GotAction)
	}
	if fc.GotParams != nil {
		t.Errorf("params = %v, want nil", fc.GotParams)
	}
	if len(list) == 0 {
		t.Errorf("len = %d, want a non-empty list", len(list))
	}
}

func TestClientGet(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "ftpuser/get_ftpuser_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	u, err := ftpuser.NewClient(fc).Get(context.Background(), "f0000001")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if fc.GotAction != "get_ftpusers" {
		t.Errorf("action = %q, want get_ftpusers", fc.GotAction)
	}
	if got, _ := fc.GotParams["ftp_login"].(string); got != "f0000001" {
		t.Errorf("params[ftp_login] = %v, want f0000001", fc.GotParams["ftp_login"])
	}
	if u.Login != "f0000001" {
		t.Errorf("Login = %q, want f0000001", u.Login)
	}
}

func TestClientGetEmptyLogin(t *testing.T) {
	t.Parallel()
	c := ftpuser.NewClient(&testutil.FakeCaller{})
	if _, err := c.Get(context.Background(), ""); err == nil {
		t.Errorf("Get(\"\") err = nil, want validation error")
	}
}

func TestClientGetNotFound(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "ftpuser/get_ftpuser_response_success_empty_list.xml")
	c := ftpuser.NewClient(&testutil.FakeCaller{Resp: resp})
	if _, err := c.Get(context.Background(), "missing"); err == nil {
		t.Errorf("Get on empty result err = nil, want not-found")
	}
}

func TestClientPropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := ftpuser.NewClient(&testutil.FakeCaller{Err: want})
	if _, err := c.List(context.Background()); !errors.Is(err, want) {
		t.Errorf("List err = %v, want %v wrapped", err, want)
	}
	if _, err := c.Get(context.Background(), "f0000001"); !errors.Is(err, want) {
		t.Errorf("Get err = %v, want %v wrapped", err, want)
	}
}

func TestFTPUserListTabular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "ftpuser/get_ftpusers_response_success.xml")
	list, _ := ftpuser.DecodeFTPUsers(resp.Body.ReturnInfo)
	headers := list.TableHeaders()
	if headers[0] != "LOGIN" {
		t.Errorf("headers[0] = %q, want LOGIN", headers[0])
	}
	rows := list.TableRows()
	if len(rows) != 5 {
		t.Fatalf("rows = %d, want 5", len(rows))
	}
	if rows[0][0] != "w0000000" {
		t.Errorf("rows[0][0] = %q, want w0000000", rows[0][0])
	}
}

func TestFTPUserTabular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "ftpuser/get_ftpuser_response_success.xml")
	list, _ := ftpuser.DecodeFTPUsers(resp.Body.ReturnInfo)
	if len(list) != 1 {
		t.Fatalf("len = %d, want 1", len(list))
	}
	u := list[0]
	headers := u.TableHeaders()
	if headers[0] != "FIELD" || headers[1] != "VALUE" {
		t.Errorf("headers = %v, want [FIELD VALUE]", headers)
	}
	rows := u.TableRows()
	if rows[0][0] != "ftp_login" || rows[0][1] != "f0000001" {
		t.Errorf("rows[0] = %v, want [ftp_login f0000001]", rows[0])
	}
}
