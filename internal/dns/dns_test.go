package dns_test

import (
	"context"
	"errors"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/dns"
	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestDecodeRecords(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "dns/get_dns_settings_response_success.xml")
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
	resp := testutil.DecodeFixture(t, "dns/get_dns_settings_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	list, err := dns.NewClient(fc).Settings(context.Background(), "example.com", "")
	if err != nil {
		t.Fatalf("Settings: %v", err)
	}
	if fc.GotAction != "get_dns_settings" {
		t.Errorf("action = %q, want get_dns_settings", fc.GotAction)
	}
	if zh, _ := fc.GotParams["zone_host"].(string); zh != "example.com" {
		t.Errorf("params[zone_host] = %v, want example.com", fc.GotParams["zone_host"])
	}
	if _, ok := fc.GotParams["record_id"]; ok {
		t.Errorf("params[record_id] set but recordID was empty: %v", fc.GotParams)
	}
	if len(list) == 0 {
		t.Errorf("len = %d, want a non-empty list", len(list))
	}
}

// With a record_id the request narrows to a single record; the
// zone_host+record_id fixture returns a one-element ReturnInfo array.
func TestClientSettingsWithRecordID(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "dns/get_dns_settings_zone_host_and_record_id_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	list, err := dns.NewClient(fc).Settings(context.Background(), "example.com", "110118416")
	if err != nil {
		t.Fatalf("Settings: %v", err)
	}
	if id, _ := fc.GotParams["record_id"].(string); id != "110118416" {
		t.Errorf("params[record_id] = %v, want 110118416", fc.GotParams["record_id"])
	}
	if len(list) != 1 || list[0].ID != "110118416" {
		t.Errorf("list = %+v, want a single record with ID 110118416", list)
	}
}

func TestClientSettingsRequiresZoneHost(t *testing.T) {
	t.Parallel()
	c := dns.NewClient(&testutil.FakeCaller{})
	if _, err := c.Settings(context.Background(), "", ""); err == nil {
		t.Errorf("Settings(\"\") err = nil, want validation error")
	}
}

func TestClientPropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := dns.NewClient(&testutil.FakeCaller{Err: want})
	if _, err := c.Settings(context.Background(), "example.com", ""); !errors.Is(err, want) {
		t.Errorf("Settings err = %v, want %v wrapped", err, want)
	}
}

func TestRecordListTabular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "dns/get_dns_settings_response_success.xml")
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
