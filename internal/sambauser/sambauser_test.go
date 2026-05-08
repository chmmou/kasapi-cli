package sambauser_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/sambauser"
	"github.com/chmmou/kasapi-cli/internal/soap"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("repo root not found from %q", file)
		}
		dir = parent
	}
}

func decodeFixture(t *testing.T, name string) *soap.Response {
	t.Helper()
	path := filepath.Join(repoRoot(t), "testdata", "sambauser", name)
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", name, err)
	}
	defer func() { _ = f.Close() }()
	resp, err := soap.Decode(f)
	if err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
	return resp
}

type fakeCaller struct {
	resp *soap.Response
	err  error

	gotAction string
	gotParams map[string]any
}

func (f *fakeCaller) Call(_ context.Context, action string, params map[string]any) (*soap.Response, error) {
	f.gotAction = action
	f.gotParams = params
	return f.resp, f.err
}

func TestDecodeSambaUsers(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_sambausers_response_success.xml")
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
	resp := decodeFixture(t, "get_sambausers_response_success.xml")
	fc := &fakeCaller{resp: resp}
	list, err := sambauser.NewClient(fc).List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if fc.gotAction != "get_sambausers" {
		t.Errorf("action = %q, want get_sambausers", fc.gotAction)
	}
	if fc.gotParams != nil {
		t.Errorf("params = %v, want nil", fc.gotParams)
	}
	if len(list) != 3 {
		t.Errorf("len = %d, want 3", len(list))
	}
}

func TestDecodeSambaUserSingular(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_sambauser_response_success.xml")
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
	resp := decodeFixture(t, "get_sambauser_response_success.xml")
	fc := &fakeCaller{resp: resp}
	u, err := sambauser.NewClient(fc).Get(context.Background(), "s0000000")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if fc.gotAction != "get_sambausers" {
		t.Errorf("action = %q, want get_sambausers", fc.gotAction)
	}
	if got, _ := fc.gotParams["samba_login"].(string); got != "s0000000" {
		t.Errorf("params[samba_login] = %v, want s0000000", fc.gotParams["samba_login"])
	}
	if u.Login != "s0000000" {
		t.Errorf("Login = %q, want s0000000", u.Login)
	}
}

func TestClientGetEmptyLogin(t *testing.T) {
	t.Parallel()
	c := sambauser.NewClient(&fakeCaller{})
	if _, err := c.Get(context.Background(), ""); err == nil {
		t.Errorf("Get(\"\") err = nil, want validation error")
	}
}

func TestClientGetNotFound(t *testing.T) {
	t.Parallel()
	resp := &soap.Response{Body: soap.ResponseBody{ReturnInfo: soap.Value{Kind: soap.KindArray}}}
	c := sambauser.NewClient(&fakeCaller{resp: resp})
	if _, err := c.Get(context.Background(), "missing"); err == nil {
		t.Errorf("Get on empty result err = nil, want not-found")
	}
}

func TestClientPropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := sambauser.NewClient(&fakeCaller{err: want})
	if _, err := c.List(context.Background()); !errors.Is(err, want) {
		t.Errorf("List err = %v, want %v wrapped", err, want)
	}
	if _, err := c.Get(context.Background(), "s0000000"); !errors.Is(err, want) {
		t.Errorf("Get err = %v, want %v wrapped", err, want)
	}
}

func TestSambaUserTabular(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_sambauser_response_success.xml")
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
	resp := decodeFixture(t, "get_sambausers_response_success.xml")
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
