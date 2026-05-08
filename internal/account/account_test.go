package account_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/account"
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
	path := filepath.Join(repoRoot(t), "testdata", "account", name)
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer func() { _ = f.Close() }()
	resp, err := soap.Decode(f)
	if err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return resp
}

func TestDecodeAccounts(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_accounts_response_success.xml")
	got, err := account.DecodeAccounts(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeAccounts: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("len = %d, want 4", len(got))
	}

	w1 := got[0]
	if w1.Login != "w0000001" {
		t.Errorf("Login = %q, want w0000001", w1.Login)
	}
	if w1.MaxAccount != 10 {
		t.Errorf("MaxAccount = %d, want 10", w1.MaxAccount)
	}
	if w1.MaxWebspace != 35600 {
		t.Errorf("MaxWebspace = %d, want 35600", w1.MaxWebspace)
	}
	if w1.UsedAccountSpace == 0 {
		t.Errorf("UsedAccountSpace not parsed: %v", w1.UsedAccountSpace)
	}
	if w1.Account2FA != "Y" {
		t.Errorf("Account2FA = %q, want Y", w1.Account2FA)
	}
	if w1.AccountContactMail != "noreply@example.org" {
		t.Errorf("AccountContactMail = %q", w1.AccountContactMail)
	}

	// w0000002 has account_2fa = "inherited" — verify it survives as a
	// string rather than being coerced to a bool somewhere.
	w2 := got[1]
	if w2.Account2FA != "inherited" {
		t.Errorf("w0000002.Account2FA = %q, want inherited", w2.Account2FA)
	}
}

func TestDecodeAccountSettings(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_accountsettings_response_success.xml")
	got, err := account.DecodeAccountSettings(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeAccountSettings: %v", err)
	}
	if got.Login != "w0000000" {
		t.Errorf("Login = %q, want w0000000", got.Login)
	}
	if got.IsSuperuser != "Y" {
		t.Errorf("IsSuperuser = %q, want Y", got.IsSuperuser)
	}
	if got.Server != "d012345" {
		t.Errorf("Server = %q", got.Server)
	}
	if got.UserPrefs.PerPage != 25 {
		t.Errorf("UserPrefs.PerPage = %d, want 25", got.UserPrefs.PerPage)
	}
	if len(got.SSHFingerprints) != 3 {
		t.Errorf("len(SSHFingerprints) = %d, want 3", len(got.SSHFingerprints))
	}
	rsa, ok := got.SSHFingerprints["RSA"]
	if !ok || rsa.SHA256 == "" {
		t.Errorf("RSA fingerprint not populated: %+v", rsa)
	}
	exp := got.UserPrefs.ExpandableTables
	if _, ok := exp["mailAccounts"]; !ok {
		t.Errorf("ExpandableTables missing mailAccounts: %v", exp)
	}
}

func TestDecodeAccountResources(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_accountresources_response_success.xml")
	got, err := account.DecodeAccountResources(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeAccountResources: %v", err)
	}
	if got.MaxSubdomain.Max != 500 {
		t.Errorf("MaxSubdomain.Max = %d, want 500", got.MaxSubdomain.Max)
	}
	if got.MaxSubdomain.Used != 15 {
		t.Errorf("MaxSubdomain.Used = %d, want 15", got.MaxSubdomain.Used)
	}
	if got.MaxDomain.Max != -1 {
		t.Errorf("MaxDomain.Max = %d, want -1 (unlimited)", got.MaxDomain.Max)
	}
	if got.MaxAccount.Free != 486 {
		t.Errorf("MaxAccount.Free = %d, want 486", got.MaxAccount.Free)
	}
	if got.MaxDatabase.Max != 50 {
		t.Errorf("MaxDatabase.Max = %d", got.MaxDatabase.Max)
	}
	if got.MaxMailingList.Max != -1 {
		t.Errorf("MaxMailingList.Max = %d, want -1", got.MaxMailingList.Max)
	}
}

// fakeCaller returns the response decoded from a fixture and records
// the action / params it was called with. This is enough to exercise
// Client.List / Get / Settings / Resources without a network
// roundtrip.
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

func TestClientList(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_accounts_response_success.xml")
	fc := &fakeCaller{resp: resp}
	got, err := account.NewClient(fc).List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if fc.gotAction != "get_accounts" {
		t.Errorf("action = %q, want get_accounts", fc.gotAction)
	}
	if fc.gotParams != nil {
		t.Errorf("params = %v, want nil", fc.gotParams)
	}
	if len(got) != 4 {
		t.Errorf("len = %d, want 4", len(got))
	}
}

func TestClientGet(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_account_response_success.xml")
	fc := &fakeCaller{resp: resp}
	got, err := account.NewClient(fc).Get(context.Background(), "w0000001")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if fc.gotAction != "get_accounts" {
		t.Errorf("action = %q, want get_accounts", fc.gotAction)
	}
	if v, ok := fc.gotParams["account_login"]; !ok || v != "w0000001" {
		t.Errorf("account_login param = %v (ok=%v), want w0000001", v, ok)
	}
	if got.Login != "w0000001" {
		t.Errorf("Login = %q, want w0000001", got.Login)
	}
	if got.MaxAccount != 10 {
		t.Errorf("MaxAccount = %d, want 10", got.MaxAccount)
	}
}

func TestClientGetEmptyLogin(t *testing.T) {
	t.Parallel()
	c := account.NewClient(&fakeCaller{})
	if _, err := c.Get(context.Background(), ""); err == nil {
		t.Error("Get(\"\") returned nil, want error")
	}
}

func TestClientGetNotFound(t *testing.T) {
	t.Parallel()
	// Synthesise an empty array response by reusing the singular
	// fixture but with the array stripped to zero entries.
	emptyResp := decodeFixture(t, "get_account_response_success.xml")
	emptyResp.Body.ReturnInfo.Array = nil
	c := account.NewClient(&fakeCaller{resp: emptyResp})
	if _, err := c.Get(context.Background(), "wXXXXXXX"); err == nil {
		t.Error("Get on empty array returned nil, want not-found error")
	}
}

func TestClientPropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := account.NewClient(&fakeCaller{err: want})
	if _, err := c.List(context.Background()); !errors.Is(err, want) {
		t.Errorf("List err = %v, want %v wrapped", err, want)
	}
	if _, err := c.Get(context.Background(), "w0000001"); !errors.Is(err, want) {
		t.Errorf("Get err = %v, want %v wrapped", err, want)
	}
}

func TestAccountListTabular(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_accounts_response_success.xml")
	accs, _ := account.DecodeAccounts(resp.Body.ReturnInfo)
	list := account.AccountList(accs)
	headers := list.TableHeaders()
	if got, want := headers[0], "LOGIN"; got != want {
		t.Errorf("headers[0] = %q, want %q", got, want)
	}
	rows := list.TableRows()
	if len(rows) != 4 {
		t.Fatalf("rows = %d, want 4", len(rows))
	}
	if rows[0][0] != "w0000001" {
		t.Errorf("rows[0][0] = %q", rows[0][0])
	}
}

func TestAccountTabular(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_account_response_success.xml")
	accs, _ := account.DecodeAccounts(resp.Body.ReturnInfo)
	if len(accs) != 1 {
		t.Fatalf("len = %d, want 1", len(accs))
	}
	a := accs[0]
	headers := a.TableHeaders()
	if headers[0] != "FIELD" || headers[1] != "VALUE" {
		t.Errorf("headers = %v, want [FIELD VALUE]", headers)
	}
	rows := a.TableRows()
	if rows[0][0] != "account_login" || rows[0][1] != "w0000001" {
		t.Errorf("rows[0] = %v, want [account_login w0000001]", rows[0])
	}
}

func TestAccountResourcesTabular(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_accountresources_response_success.xml")
	r, _ := account.DecodeAccountResources(resp.Body.ReturnInfo)
	rows := r.TableRows()
	if len(rows) != 12 {
		t.Errorf("rows = %d, want 12", len(rows))
	}
	// max_domain has Max = -1 → "∞"
	for _, row := range rows {
		if row[0] == "max_domain" && row[1] != "∞" {
			t.Errorf("max_domain.Max = %q, want ∞", row[1])
		}
	}
}
