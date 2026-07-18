package ddns_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/ddns"
	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestDecodeDDNSUsers(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "ddns/get_ddnsusers_response_success.xml")
	got, err := ddns.DecodeDDNSUsers(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeDDNSUsers: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	first := got[0]
	if first.Login != "dyn0000002" {
		t.Errorf("Login = %q, want dyn0000002", first.Login)
	}
	if first.Zone != "example.org" || first.Label != "test2" {
		t.Errorf("zone/label = %q/%q", first.Zone, first.Label)
	}
	if first.FQDN() != "test2.example.org" {
		t.Errorf("FQDN = %q, want test2.example.org", first.FQDN())
	}
	if first.TargetIP != "127.0.0.255" {
		t.Errorf("TargetIP = %q", first.TargetIP)
	}
	// The list endpoint also surfaces target_ipv4 / target_ipv6.
	if first.TargetIPv4 != "127.0.0.255" {
		t.Errorf("TargetIPv4 = %q", first.TargetIPv4)
	}
	if first.TargetIPv6 == "" {
		t.Errorf("TargetIPv6 not populated on list entry")
	}
	if first.DualStack != "Y" {
		t.Errorf("DualStack = %q, want Y", first.DualStack)
	}
	if got[1].Login != "dyn0000001" {
		t.Errorf("got[1].Login = %q, want dyn0000001", got[1].Login)
	}
	if got[1].DualStack != "N" {
		t.Errorf("got[1].DualStack = %q, want N", got[1].DualStack)
	}
}

func TestDecodeDDNSUserSingular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "ddns/get_ddnsusers_response_success_single.xml")
	got, err := ddns.DecodeDDNSUsers(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeDDNSUsers: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	u := got[0]
	if u.Login != "dyn0000002" {
		t.Errorf("Login = %q, want dyn0000002", u.Login)
	}
	if u.FQDN() != "test2.example.org" {
		t.Errorf("FQDN = %q, want test2.example.org", u.FQDN())
	}
	// The singular variant carries the explicit ipv4/ipv6 fields
	// alongside the legacy target_ip; both must round-trip.
	if u.TargetIPv4 != "127.0.0.255" {
		t.Errorf("TargetIPv4 = %q", u.TargetIPv4)
	}
	if u.TargetIPv6 != "2001:0db8:85a3:0000:0000:8a2e:0370:7334" {
		t.Errorf("TargetIPv6 = %q", u.TargetIPv6)
	}
	// in_progress is absent on the singular variant — must decode
	// to the empty string, not panic or default to anything else.
	if u.InProgress != "" {
		t.Errorf("InProgress = %q, want empty (key absent in fixture)", u.InProgress)
	}
}

func TestFQDN(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   ddns.DDNSUser
		want string
	}{
		{ddns.DDNSUser{Label: "home", Zone: "example.com"}, "home.example.com"},
		{ddns.DDNSUser{Label: "", Zone: "example.com"}, "example.com"},
		{ddns.DDNSUser{Label: "home", Zone: ""}, "home"},
	}
	for _, tc := range cases {
		if got := tc.in.FQDN(); got != tc.want {
			t.Errorf("FQDN(%+v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestClientList(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "ddns/get_ddnsusers_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	list, err := ddns.NewClient(fc).List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if fc.GotAction != "get_ddnsusers" {
		t.Errorf("action = %q, want get_ddnsusers", fc.GotAction)
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
	resp := testutil.DecodeFixture(t, "ddns/get_ddnsusers_response_success_single.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	u, err := ddns.NewClient(fc).Get(context.Background(), "dyn0000002")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if fc.GotAction != "get_ddnsusers" {
		t.Errorf("action = %q", fc.GotAction)
	}
	// Filter parameter is `ddns_login` (no y), unlike the response
	// keys which use the dyndns_ prefix. Fixture-confirmed.
	if got, _ := fc.GotParams["ddns_login"].(string); got != "dyn0000002" {
		t.Errorf("params[ddns_login] = %v, want dyn0000002", fc.GotParams["ddns_login"])
	}
	if _, ok := fc.GotParams["dyndns_login"]; ok {
		t.Errorf("dyndns_login (with y) leaked into params: %v", fc.GotParams)
	}
	if u.Login != "dyn0000002" {
		t.Errorf("Login = %q", u.Login)
	}
}

func TestClientGetEmptyLogin(t *testing.T) {
	t.Parallel()
	c := ddns.NewClient(&testutil.FakeCaller{})
	if _, err := c.Get(context.Background(), ""); err == nil {
		t.Errorf("Get(\"\") err = nil, want validation error")
	}
}

// TestClientGetNotFound covers the empty-array fallback in
// kasread.ListGet.Get. The KAS docs say a missing dyndns_login is
// signalled via a SOAP fault (dyndns_login_not_found) rather than
// an empty array, but ddns.Get carries a defensive len(list) == 0
// branch for parity with the other read modules; this test pins
// that branch's behaviour by feeding the singular fixture with the
// array stripped down to zero entries.
func TestClientGetNotFound(t *testing.T) {
	t.Parallel()
	emptyResp := testutil.DecodeFixture(t, "ddns/get_ddnsusers_response_success_single.xml")
	emptyResp.Body.ReturnInfo.Array = nil
	c := ddns.NewClient(&testutil.FakeCaller{Resp: emptyResp})
	_, err := c.Get(context.Background(), "ghost")
	if err == nil {
		t.Fatal("Get on empty array returned nil, want not-found error")
	}
	if !strings.Contains(err.Error(), `"ghost" not found`) {
		t.Errorf("err = %q, want it to contain '%q not found'", err, "ghost")
	}
}

func TestClientPropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := ddns.NewClient(&testutil.FakeCaller{Err: want})
	if _, err := c.List(context.Background()); !errors.Is(err, want) {
		t.Errorf("List err = %v, want %v wrapped", err, want)
	}
	if _, err := c.Get(context.Background(), "dyn0000002"); !errors.Is(err, want) {
		t.Errorf("Get err = %v, want %v wrapped", err, want)
	}
}

func TestDDNSUserListTabular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "ddns/get_ddnsusers_response_success.xml")
	list, _ := ddns.DecodeDDNSUsers(resp.Body.ReturnInfo)
	headers := list.TableHeaders()
	if headers[0] != "LOGIN" {
		t.Errorf("headers[0] = %q, want LOGIN", headers[0])
	}
	rows := list.TableRows()
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	if rows[0][0] != "dyn0000002" {
		t.Errorf("rows[0][LOGIN] = %q, want dyn0000002", rows[0][0])
	}
	if rows[0][1] != "test2.example.org" {
		t.Errorf("rows[0][FQDN] = %q, want test2.example.org", rows[0][1])
	}
}

func TestDDNSUserTabular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "ddns/get_ddnsusers_response_success_single.xml")
	list, _ := ddns.DecodeDDNSUsers(resp.Body.ReturnInfo)
	if len(list) != 1 {
		t.Fatalf("len = %d, want 1", len(list))
	}
	u := list[0]
	headers := u.TableHeaders()
	if headers[0] != "FIELD" || headers[1] != "VALUE" {
		t.Errorf("headers = %v, want [FIELD VALUE]", headers)
	}
	rows := u.TableRows()
	if rows[0][0] != "dyndns_login" || rows[0][1] != "dyn0000002" {
		t.Errorf("rows[0] = %v, want [dyndns_login dyn0000002]", rows[0])
	}
	// dyndns_password must NOT appear in the K/V table; the
	// optional ipv4/ipv6 detail rows must be present because the
	// singular fixture carries them.
	have := map[string]bool{}
	for _, row := range rows {
		if row[0] == "dyndns_password" {
			t.Errorf("dyndns_password leaked into table view: %v", row)
		}
		have[row[0]] = true
	}
	for _, want := range []string{"dyndns_target_ipv4", "dyndns_target_ipv6"} {
		if !have[want] {
			t.Errorf("singular table missing %q row", want)
		}
	}
}
