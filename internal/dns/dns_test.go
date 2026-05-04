package dns_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/dns"
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
	path := filepath.Join(repoRoot(t), "testdata", "dns", name)
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

func TestDecodeRecords(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_dns_settings_response_success.xml")
	got, err := dns.DecodeRecords(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeRecords: %v", err)
	}
	if len(got) != 6 {
		t.Fatalf("len = %d, want 6", len(got))
	}

	// First record: A record at apex pointing at 127.0.0.1.
	a := got[0]
	if a.Type != "A" || a.Data != "127.0.0.1" || a.Name != "" {
		t.Errorf("first record = %+v", a)
	}
	if a.Aux != 0 || a.ID != "22675113" {
		t.Errorf("first record aux/id = %d/%q", a.Aux, a.ID)
	}
	if a.Changeable != "Y" || a.Deleteable != "Y" {
		t.Errorf("first record flags = %q/%q", a.Changeable, a.Deleteable)
	}

	// Second record: MX with aux=10.
	mx := got[1]
	if mx.Type != "MX" || mx.Aux != 10 {
		t.Errorf("MX record aux = %d, want 10", mx.Aux)
	}

	// DKIM TXT entry has Deleteable=N (the only non-Y flag in the
	// fixture); make sure that round-trips.
	dkim := got[5]
	if dkim.Name != "abc012345678901._domainkey" {
		t.Errorf("dkim record name = %q", dkim.Name)
	}
	if dkim.Deleteable != "N" {
		t.Errorf("dkim Deleteable = %q, want N", dkim.Deleteable)
	}
}

func TestClientSettings(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_dns_settings_response_success.xml")
	fc := &fakeCaller{resp: resp}
	list, err := dns.NewClient(fc).Settings(context.Background(), "example.com", "")
	if err != nil {
		t.Fatalf("Settings: %v", err)
	}
	if fc.gotAction != "get_dns_settings" {
		t.Errorf("action = %q, want get_dns_settings", fc.gotAction)
	}
	if zh, _ := fc.gotParams["zone_host"].(string); zh != "example.com" {
		t.Errorf("params[zone_host] = %v, want example.com", fc.gotParams["zone_host"])
	}
	if _, ok := fc.gotParams["nameserver"]; ok {
		t.Errorf("params[nameserver] set but nameserver was empty: %v", fc.gotParams)
	}
	if len(list) != 6 {
		t.Errorf("len = %d, want 6", len(list))
	}
}

func TestClientSettingsWithNameserver(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_dns_settings_response_success.xml")
	fc := &fakeCaller{resp: resp}
	if _, err := dns.NewClient(fc).Settings(context.Background(), "example.com", "ns.example.com"); err != nil {
		t.Fatalf("Settings: %v", err)
	}
	if ns, _ := fc.gotParams["nameserver"].(string); ns != "ns.example.com" {
		t.Errorf("params[nameserver] = %v, want ns.example.com", fc.gotParams["nameserver"])
	}
}

func TestClientSettingsRequiresZoneHost(t *testing.T) {
	t.Parallel()
	c := dns.NewClient(&fakeCaller{})
	if _, err := c.Settings(context.Background(), "", ""); err == nil {
		t.Errorf("Settings(\"\") err = nil, want validation error")
	}
}

func TestClientPropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := dns.NewClient(&fakeCaller{err: want})
	if _, err := c.Settings(context.Background(), "example.com", ""); !errors.Is(err, want) {
		t.Errorf("Settings err = %v, want %v wrapped", err, want)
	}
}

func TestRecordListTabular(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_dns_settings_response_success.xml")
	list, _ := dns.DecodeRecords(resp.Body.ReturnInfo)
	rows := list.TableRows()
	if len(rows) != 6 {
		t.Fatalf("rows = %d, want 6", len(rows))
	}
	if rows[0][0] != "22675113" || rows[0][3] != "A" {
		t.Errorf("rows[0] = %v", rows[0])
	}
	if rows[1][3] != "MX" || rows[1][4] != "10" {
		t.Errorf("rows[1] aux column = %v", rows[1])
	}
}
