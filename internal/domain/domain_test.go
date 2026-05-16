package domain_test

import (
	"context"
	"errors"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/domain"
	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestDecodeDomains(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "domain/get_domains_response_success.xml")
	got, err := domain.DecodeDomains(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeDomains: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	d := got[0]
	if d.Name != "example.com" || d.TLD != "com" {
		t.Errorf("first domain = %q/%q, want example.com/com", d.Name, d.TLD)
	}
	if d.Account != "w0000000" || d.Server != "ab12345" {
		t.Errorf("first placement = %q/%q", d.Account, d.Server)
	}
	if d.PHPVersion != "8.4" {
		t.Errorf("PHPVersion = %q", d.PHPVersion)
	}
	if d.IsActive != "Y" {
		t.Errorf("IsActive = %q", d.IsActive)
	}
	if d.SSL.SNI != "Y" || d.SSL.SNIType != "unknown" {
		t.Errorf("SSL SNI = %+v", d.SSL)
	}
	// In the list view the cert PEM bodies are xsi:nil → empty string.
	if d.SSL.SNICRT != "" || d.SSL.SNIKey != "" {
		t.Errorf("list view leaks cert PEM: crt=%q key=%q", d.SSL.SNICRT, d.SSL.SNIKey)
	}
}

func TestDecodeDomainSingular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "domain/get_domain_response_success.xml")
	got, err := domain.DecodeDomains(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeDomains: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	d := got[0]
	if d.Name != "example.com" {
		t.Errorf("Name = %q", d.Name)
	}
	if d.DKIMSelector != "abc190101010000" {
		t.Errorf("DKIMSelector = %q", d.DKIMSelector)
	}
	if d.DummyHost != "N" {
		t.Errorf("DummyHost = %q", d.DummyHost)
	}
	if d.SSL.SNIType != "LE90D" {
		t.Errorf("SNIType = %q, want LE90D", d.SSL.SNIType)
	}
	if d.SSL.SNIForceHTTPS != "Y" {
		t.Errorf("SNIForceHTTPS = %q", d.SSL.SNIForceHTTPS)
	}
	if d.SSL.SNIHSTSMaxAge != "-1" {
		t.Errorf("SNIHSTSMaxAge = %q", d.SSL.SNIHSTSMaxAge)
	}
	// The singular view fills the cert PEM bodies; verify they are
	// non-empty so the SSL.SNI* round-trip works.
	if d.SSL.SNICRT == "" || d.SSL.SNIKey == "" || d.SSL.SNICSR == "" {
		t.Errorf("singular view missing cert PEM body: csr=%q key=%q crt=%q",
			d.SSL.SNICSR, d.SSL.SNIKey, d.SSL.SNICRT)
	}
}

func TestDecodeTLDs(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "domain/get_topleveldomains_response_success.xml")
	got, err := domain.DecodeTLDs(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeTLDs: %v", err)
	}
	if len(got) != 1077 {
		t.Fatalf("len = %d, want 1077 (per fixture arrayType)", len(got))
	}
	first := got[0]
	if first.Name != "de" || first.MinLen != 1 || first.MaxLen != 63 {
		t.Errorf("first = %+v, want {de,1,63}", first)
	}
	// Spot-check a TLD with min length > 1 to confirm the xsd:string
	// → int parse path actually runs.
	for _, tld := range got {
		if tld.Name == "info" {
			if tld.MinLen != 3 {
				t.Errorf("info.MinLen = %d, want 3", tld.MinLen)
			}
			break
		}
	}
}

func TestClientList(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "domain/get_domains_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	list, err := domain.NewClient(fc).List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if fc.GotAction != "get_domains" {
		t.Errorf("action = %q, want get_domains", fc.GotAction)
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
	resp := testutil.DecodeFixture(t, "domain/get_domain_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	d, err := domain.NewClient(fc).Get(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if fc.GotAction != "get_domains" {
		t.Errorf("action = %q, want get_domains", fc.GotAction)
	}
	if name, _ := fc.GotParams["domain_name"].(string); name != "example.com" {
		t.Errorf("params[domain_name] = %v, want example.com", fc.GotParams["domain_name"])
	}
	if d.Name != "example.com" {
		t.Errorf("Name = %q", d.Name)
	}
}

func TestClientGetEmptyName(t *testing.T) {
	t.Parallel()
	c := domain.NewClient(&testutil.FakeCaller{})
	if _, err := c.Get(context.Background(), ""); err == nil {
		t.Errorf("Get(\"\") err = nil, want validation error")
	}
}

func TestClientGetNotFound(t *testing.T) {
	t.Parallel()
	// Empty array fixture: re-use get_domains_request via a hand-made
	// soap.Response with a zero-length array.
	resp := &soap.Response{Body: soap.ResponseBody{ReturnInfo: soap.Value{Kind: soap.KindArray}}}
	c := domain.NewClient(&testutil.FakeCaller{Resp: resp})
	if _, err := c.Get(context.Background(), "missing.example"); err == nil {
		t.Errorf("Get on empty result err = nil, want not-found")
	}
}

func TestClientTopLevelDomains(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "domain/get_topleveldomains_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	list, err := domain.NewClient(fc).TopLevelDomains(context.Background())
	if err != nil {
		t.Fatalf("TopLevelDomains: %v", err)
	}
	if fc.GotAction != "get_topleveldomains" {
		t.Errorf("action = %q, want get_topleveldomains", fc.GotAction)
	}
	if len(list) == 0 {
		t.Errorf("len = %d, want a non-empty list", len(list))
	}
}

func TestClientPropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := domain.NewClient(&testutil.FakeCaller{Err: want})
	if _, err := c.List(context.Background()); !errors.Is(err, want) {
		t.Errorf("List err = %v, want %v wrapped", err, want)
	}
	if _, err := c.Get(context.Background(), "x.example"); !errors.Is(err, want) {
		t.Errorf("Get err = %v, want %v wrapped", err, want)
	}
	if _, err := c.TopLevelDomains(context.Background()); !errors.Is(err, want) {
		t.Errorf("TopLevelDomains err = %v, want %v wrapped", err, want)
	}
}

func TestDomainListTabular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "domain/get_domains_response_success.xml")
	list, _ := domain.DecodeDomains(resp.Body.ReturnInfo)
	rows := list.TableRows()
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	if rows[0][0] != "example.com" || rows[0][1] != "w0000000" {
		t.Errorf("rows[0] = %v", rows[0])
	}
}

func TestTLDListTabular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "domain/get_topleveldomains_response_success.xml")
	list, _ := domain.DecodeTLDs(resp.Body.ReturnInfo)
	rows := list.TableRows()
	if len(rows) != 1077 {
		t.Fatalf("rows = %d, want 1077", len(rows))
	}
	if rows[0][0] != "de" || rows[0][1] != "1" || rows[0][2] != "63" {
		t.Errorf("rows[0] = %v", rows[0])
	}
}

func TestDomainTabularSummarisesPEM(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "domain/get_domain_response_success.xml")
	list, _ := domain.DecodeDomains(resp.Body.ReturnInfo)
	rows := list[0].TableRows()
	for _, row := range rows {
		if row[0] == "ssl_certificate_sni_crt" && row[1] != "" {
			// Multi-line PEM blob must be summarised, not pasted in.
			if row[1][0] != '<' {
				t.Errorf("ssl_certificate_sni_crt row not summarised: %q", row[1])
			}
		}
	}
}
