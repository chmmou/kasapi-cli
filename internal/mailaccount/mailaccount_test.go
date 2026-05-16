package mailaccount_test

import (
	"context"
	"errors"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/mailaccount"
	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestDecodeMailAccounts(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "mailaccount/get_mailaccounts_response_success.xml")
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
	resp := testutil.DecodeFixture(t, "mailaccount/get_mailaccount_response_success.xml")
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
	resp := testutil.DecodeFixture(t, "mailaccount/get_mailaccounts_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	list, err := mailaccount.NewClient(fc).List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if fc.GotAction != "get_mailaccounts" {
		t.Errorf("action = %q, want get_mailaccounts", fc.GotAction)
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
	resp := testutil.DecodeFixture(t, "mailaccount/get_mailaccount_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	a, err := mailaccount.NewClient(fc).Get(context.Background(), "m0000001")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if fc.GotAction != "get_mailaccounts" {
		t.Errorf("action = %q, want get_mailaccounts", fc.GotAction)
	}
	if got, _ := fc.GotParams["mail_login"].(string); got != "m0000001" {
		t.Errorf("params[mail_login] = %v, want m0000001", fc.GotParams["mail_login"])
	}
	// The returned row must echo the requested login. The singular
	// fixture's request echo and response row are aligned, so this can
	// assert equality rather than just non-emptiness.
	if a.Login != "m0000001" {
		t.Errorf("Login = %q, want m0000001 (the requested login)", a.Login)
	}
}

func TestClientGetEmptyLogin(t *testing.T) {
	t.Parallel()
	c := mailaccount.NewClient(&testutil.FakeCaller{})
	if _, err := c.Get(context.Background(), ""); err == nil {
		t.Errorf("Get(\"\") err = nil, want validation error")
	}
}

func TestClientGetNotFound(t *testing.T) {
	t.Parallel()
	resp := &soap.Response{Body: soap.ResponseBody{ReturnInfo: soap.Value{Kind: soap.KindArray}}}
	c := mailaccount.NewClient(&testutil.FakeCaller{Resp: resp})
	if _, err := c.Get(context.Background(), "missing"); err == nil {
		t.Errorf("Get on empty result err = nil, want not-found")
	}
}

func TestClientPropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := mailaccount.NewClient(&testutil.FakeCaller{Err: want})
	if _, err := c.List(context.Background()); !errors.Is(err, want) {
		t.Errorf("List err = %v, want %v wrapped", err, want)
	}
	if _, err := c.Get(context.Background(), "m0000001"); !errors.Is(err, want) {
		t.Errorf("Get err = %v, want %v wrapped", err, want)
	}
}

func TestMailAccountListTabular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "mailaccount/get_mailaccounts_response_success.xml")
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
	resp := testutil.DecodeFixture(t, "mailaccount/get_mailaccount_response_success.xml")
	list, _ := mailaccount.DecodeMailAccounts(resp.Body.ReturnInfo)
	rows := list[0].TableRows()
	if len(rows) == 0 {
		t.Fatal("no rows")
	}
	if rows[0][0] != "mail_login" {
		t.Errorf("rows[0][0] = %q, want mail_login", rows[0][0])
	}
}
