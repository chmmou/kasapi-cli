package mailaccount_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/mailaccount"
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
	path := filepath.Join(repoRoot(t), "testdata", "mailaccount", name)
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

func TestDecodeMailAccounts(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_mailaccounts_response_success.xml")
	got, err := mailaccount.DecodeMailAccounts(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeMailAccounts: %v", err)
	}
	if len(got) != 15 {
		t.Fatalf("len = %d, want 15 (per fixture arrayType)", len(got))
	}
	a := got[0]
	if a.Login != "m0000001" {
		t.Errorf("Login = %q, want m0000001", a.Login)
	}
	if a.Addresses != "m0000001@example.com" || a.Adresses != "m0000001@example.com" {
		t.Errorf("addresses pair = %q / %q", a.Addresses, a.Adresses)
	}
	if a.Spamfilter != "pdw,sf" {
		t.Errorf("Spamfilter = %q", a.Spamfilter)
	}
	if a.UsedSpace == 0 {
		t.Errorf("UsedSpace = 0, want non-zero from xsd:float")
	}
	if a.IsActive != "Y" {
		t.Errorf("IsActive = %q", a.IsActive)
	}
	if a.WebmailAutologin != "Y" {
		t.Errorf("WebmailAutologin = %q", a.WebmailAutologin)
	}
}

func TestDecodeMailAccountSingular(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_mailaccount_response_success.xml")
	got, err := mailaccount.DecodeMailAccounts(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeMailAccounts: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	a := got[0]
	if a.Login == "" {
		t.Errorf("Login empty")
	}
	if a.UsedSpace == 0 {
		t.Errorf("UsedSpace = 0, want non-zero")
	}
}

func TestClientList(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_mailaccounts_response_success.xml")
	fc := &fakeCaller{resp: resp}
	list, err := mailaccount.NewClient(fc).List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if fc.gotAction != "get_mailaccounts" {
		t.Errorf("action = %q, want get_mailaccounts", fc.gotAction)
	}
	if fc.gotParams != nil {
		t.Errorf("params = %v, want nil", fc.gotParams)
	}
	if len(list) != 15 {
		t.Errorf("len = %d, want 15", len(list))
	}
}

func TestClientGet(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_mailaccount_response_success.xml")
	fc := &fakeCaller{resp: resp}
	a, err := mailaccount.NewClient(fc).Get(context.Background(), "m0000001")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if fc.gotAction != "get_mailaccounts" {
		t.Errorf("action = %q, want get_mailaccounts", fc.gotAction)
	}
	if got, _ := fc.gotParams["mail_login"].(string); got != "m0000001" {
		t.Errorf("params[mail_login] = %v, want m0000001", fc.gotParams["mail_login"])
	}
	if a.Login == "" {
		t.Errorf("Login empty")
	}
}

func TestClientGetEmptyLogin(t *testing.T) {
	t.Parallel()
	c := mailaccount.NewClient(&fakeCaller{})
	if _, err := c.Get(context.Background(), ""); err == nil {
		t.Errorf("Get(\"\") err = nil, want validation error")
	}
}

func TestClientGetNotFound(t *testing.T) {
	t.Parallel()
	resp := &soap.Response{Body: soap.ResponseBody{ReturnInfo: soap.Value{Kind: soap.KindArray}}}
	c := mailaccount.NewClient(&fakeCaller{resp: resp})
	if _, err := c.Get(context.Background(), "missing"); err == nil {
		t.Errorf("Get on empty result err = nil, want not-found")
	}
}

func TestClientPropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := mailaccount.NewClient(&fakeCaller{err: want})
	if _, err := c.List(context.Background()); !errors.Is(err, want) {
		t.Errorf("List err = %v, want %v wrapped", err, want)
	}
	if _, err := c.Get(context.Background(), "m0000001"); !errors.Is(err, want) {
		t.Errorf("Get err = %v, want %v wrapped", err, want)
	}
}

func TestMailAccountListTabular(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_mailaccounts_response_success.xml")
	list, _ := mailaccount.DecodeMailAccounts(resp.Body.ReturnInfo)
	rows := list.TableRows()
	if len(rows) != 15 {
		t.Fatalf("rows = %d, want 15", len(rows))
	}
	if rows[0][0] != "m0000001" {
		t.Errorf("rows[0][0] = %q, want m0000001", rows[0][0])
	}
}

func TestMailAccountTabular(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_mailaccount_response_success.xml")
	list, _ := mailaccount.DecodeMailAccounts(resp.Body.ReturnInfo)
	rows := list[0].TableRows()
	if len(rows) == 0 {
		t.Fatal("no rows")
	}
	if rows[0][0] != "mail_login" {
		t.Errorf("rows[0][0] = %q, want mail_login", rows[0][0])
	}
}
