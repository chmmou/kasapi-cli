package subdomain_test

import (
	"context"
	"errors"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/subdomain"
	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestDecodeSubdomains(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "subdomain/get_subdomains_response_success.xml")
	got, err := subdomain.DecodeSubdomains(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeSubdomains: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	first := got[0]
	if first.Name != "sub1.example.com" {
		t.Errorf("Name = %q", first.Name)
	}
	if first.Account != "w0000000" || first.Server != "ab12345" {
		t.Errorf("placement = %q/%q", first.Account, first.Server)
	}
	if first.PHPVersion != "8.4" || first.IsActive != "Y" {
		t.Errorf("first php/active = %q/%q", first.PHPVersion, first.IsActive)
	}
	if first.SSL.SNI != "Y" || first.SSL.SNIType != "unknown" {
		t.Errorf("first SSL = %+v", first.SSL)
	}
}

func TestClientList(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "subdomain/get_subdomains_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	list, err := subdomain.NewClient(fc).List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if fc.GotAction != "get_subdomains" {
		t.Errorf("action = %q, want get_subdomains", fc.GotAction)
	}
	if fc.GotParams != nil {
		t.Errorf("params = %v, want nil", fc.GotParams)
	}
	if len(list) != 2 {
		t.Errorf("len = %d, want 2", len(list))
	}
}

func TestClientGet(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "subdomain/get_subdomains_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	s, err := subdomain.NewClient(fc).Get(context.Background(), "sub1.example.com")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if fc.GotAction != "get_subdomains" {
		t.Errorf("action = %q, want get_subdomains", fc.GotAction)
	}
	if name, _ := fc.GotParams["subdomain_name"].(string); name != "sub1.example.com" {
		t.Errorf("params[subdomain_name] = %v, want sub1.example.com", fc.GotParams["subdomain_name"])
	}
	// The fixture is the list-view payload; Get unwraps the first entry.
	if s.Name != "sub1.example.com" {
		t.Errorf("Name = %q", s.Name)
	}
}

func TestClientGetEmptyName(t *testing.T) {
	t.Parallel()
	c := subdomain.NewClient(&testutil.FakeCaller{})
	if _, err := c.Get(context.Background(), ""); err == nil {
		t.Errorf("Get(\"\") err = nil, want validation error")
	}
}

func TestClientGetNotFound(t *testing.T) {
	t.Parallel()
	resp := &soap.Response{Body: soap.ResponseBody{ReturnInfo: soap.Value{Kind: soap.KindArray}}}
	c := subdomain.NewClient(&testutil.FakeCaller{Resp: resp})
	if _, err := c.Get(context.Background(), "missing.example.com"); err == nil {
		t.Errorf("Get on empty result err = nil, want not-found")
	}
}

func TestClientPropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := subdomain.NewClient(&testutil.FakeCaller{Err: want})
	if _, err := c.List(context.Background()); !errors.Is(err, want) {
		t.Errorf("List err = %v, want %v wrapped", err, want)
	}
	if _, err := c.Get(context.Background(), "x.example.com"); !errors.Is(err, want) {
		t.Errorf("Get err = %v, want %v wrapped", err, want)
	}
}

func TestSubdomainTabularKeyValue(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "subdomain/get_subdomains_response_success.xml")
	list, _ := subdomain.DecodeSubdomains(resp.Body.ReturnInfo)
	if len(list) == 0 {
		t.Fatal("fixture empty")
	}
	headers := list[0].TableHeaders()
	if len(headers) != 2 || headers[0] != "FIELD" || headers[1] != "VALUE" {
		t.Errorf("headers = %v, want [FIELD VALUE]", headers)
	}
	rows := list[0].TableRows()
	var seen bool
	for _, row := range rows {
		if row[0] == "subdomain_name" && row[1] == "sub1.example.com" {
			seen = true
		}
	}
	if !seen {
		t.Errorf("subdomain_name row missing or wrong: %v", rows)
	}
}

func TestSubdomainListTabular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "subdomain/get_subdomains_response_success.xml")
	list, _ := subdomain.DecodeSubdomains(resp.Body.ReturnInfo)
	rows := list.TableRows()
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	if rows[0][0] != "sub1.example.com" || rows[0][1] != "w0000000" {
		t.Errorf("rows[0] = %v", rows[0])
	}
}
