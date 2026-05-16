package sambauser_test

import (
	"context"
	"errors"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/sambauser"
	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestDecodeSambaUsers(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "sambauser/get_sambausers_response_success.xml")
	got, err := sambauser.DecodeSambaUsers(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeSambaUsers: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	first := got[0]
	if first.Login != "s0000000" {
		t.Errorf("Login = %q, want s0000000", first.Login)
	}
	if first.Path != "/example.com/share/" {
		t.Errorf("Path = %q", first.Path)
	}
	if first.Comment != "Shared documents" {
		t.Errorf("Comment = %q", first.Comment)
	}
	if first.InProgress != "FALSE" {
		t.Errorf("InProgress = %q, want FALSE", first.InProgress)
	}
}

func TestClientList(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "sambauser/get_sambausers_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	list, err := sambauser.NewClient(fc).List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if fc.GotAction != "get_sambausers" {
		t.Errorf("action = %q, want get_sambausers", fc.GotAction)
	}
	if fc.GotParams != nil {
		t.Errorf("params = %v, want nil", fc.GotParams)
	}
	if len(list) == 0 {
		t.Errorf("len = %d, want a non-empty list", len(list))
	}
}

func TestDecodeSambaUserSingular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "sambauser/get_sambauser_response_success.xml")
	got, err := sambauser.DecodeSambaUsers(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeSambaUsers: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	u := got[0]
	if u.Login != "s0000000" {
		t.Errorf("Login = %q, want s0000000", u.Login)
	}
	if u.Path != "/example.com/share/" {
		t.Errorf("Path = %q", u.Path)
	}
}

func TestClientGet(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "sambauser/get_sambauser_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	u, err := sambauser.NewClient(fc).Get(context.Background(), "s0000000")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if fc.GotAction != "get_sambausers" {
		t.Errorf("action = %q, want get_sambausers", fc.GotAction)
	}
	if got, _ := fc.GotParams["samba_login"].(string); got != "s0000000" {
		t.Errorf("params[samba_login] = %v, want s0000000", fc.GotParams["samba_login"])
	}
	if u.Login != "s0000000" {
		t.Errorf("Login = %q, want s0000000", u.Login)
	}
}

func TestClientGetEmptyLogin(t *testing.T) {
	t.Parallel()
	c := sambauser.NewClient(&testutil.FakeCaller{})
	if _, err := c.Get(context.Background(), ""); err == nil {
		t.Errorf("Get(\"\") err = nil, want validation error")
	}
}

func TestClientGetNotFound(t *testing.T) {
	t.Parallel()
	resp := &soap.Response{Body: soap.ResponseBody{ReturnInfo: soap.Value{Kind: soap.KindArray}}}
	c := sambauser.NewClient(&testutil.FakeCaller{Resp: resp})
	if _, err := c.Get(context.Background(), "missing"); err == nil {
		t.Errorf("Get on empty result err = nil, want not-found")
	}
}

func TestClientPropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := sambauser.NewClient(&testutil.FakeCaller{Err: want})
	if _, err := c.List(context.Background()); !errors.Is(err, want) {
		t.Errorf("List err = %v, want %v wrapped", err, want)
	}
	if _, err := c.Get(context.Background(), "s0000000"); !errors.Is(err, want) {
		t.Errorf("Get err = %v, want %v wrapped", err, want)
	}
}

func TestSambaUserTabular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "sambauser/get_sambauser_response_success.xml")
	list, _ := sambauser.DecodeSambaUsers(resp.Body.ReturnInfo)
	if len(list) != 1 {
		t.Fatalf("len = %d, want 1", len(list))
	}
	u := list[0]
	headers := u.TableHeaders()
	if headers[0] != "FIELD" || headers[1] != "VALUE" {
		t.Errorf("headers = %v, want [FIELD VALUE]", headers)
	}
	rows := u.TableRows()
	if rows[0][0] != "samba_login" || rows[0][1] != "s0000000" {
		t.Errorf("rows[0] = %v, want [samba_login s0000000]", rows[0])
	}
}

func TestSambaUserListTabular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "sambauser/get_sambausers_response_success.xml")
	list, _ := sambauser.DecodeSambaUsers(resp.Body.ReturnInfo)
	headers := list.TableHeaders()
	if headers[0] != "LOGIN" {
		t.Errorf("headers[0] = %q, want LOGIN", headers[0])
	}
	rows := list.TableRows()
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want 3", len(rows))
	}
	if rows[0][0] != "s0000000" {
		t.Errorf("rows[0][0] = %q, want s0000000", rows[0][0])
	}
}
