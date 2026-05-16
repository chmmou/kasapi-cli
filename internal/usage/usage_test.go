package usage_test

import (
	"context"
	"errors"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/testutil"
	"github.com/chmmou/kasapi-cli/internal/usage"
)

// trafficRow builds a get_traffic ReturnInfo map with a single entry
// whose year/month carry the given raw string values, so the strict
// decode of the mandatory period fields can be exercised in isolation.
func trafficRow(year, month string) soap.Value {
	return soap.Value{Kind: soap.KindMap, Map: []soap.KV{
		{Key: "0", Value: soap.Value{Kind: soap.KindMap, Map: []soap.KV{
			{Key: "account_login", Value: soap.Value{Kind: soap.KindString, String: "w0000001"}},
			{Key: "year", Value: soap.Value{Kind: soap.KindString, String: year}},
			{Key: "month", Value: soap.Value{Kind: soap.KindString, String: month}},
		}}},
	}}
}

func TestDecodeSpace(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "usage/get_space_response_success.xml")
	got, err := usage.DecodeSpace(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeSpace: %v", err)
	}
	if len(got) != 5 {
		t.Fatalf("len = %d, want 5", len(got))
	}
	first := got[0]
	if first.AccountLogin != "w0000001" {
		t.Errorf("first.AccountLogin = %q, want w0000001", first.AccountLogin)
	}
	if first.UsedWebspace != 11947780 || first.MaxWebspace != 36454400 {
		t.Errorf("first webspace = %d/%d, want 11947780/36454400",
			first.UsedWebspace, first.MaxWebspace)
	}
	if first.UsedHTDocsSpace != 4843107 || first.UsedMailaccountSpace != 6877004 {
		t.Errorf("first detail bytes = htdocs %d, mail %d",
			first.UsedHTDocsSpace, first.UsedMailaccountSpace)
	}
	if first.LastCalculation != 1777688517 {
		t.Errorf("first.LastCalculation = %d, want 1777688517", first.LastCalculation)
	}
}

func TestDecodeSpaceUsage(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "usage/get_space_usage_response_success.xml")
	got, err := usage.DecodeSpaceUsage(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeSpaceUsage: %v", err)
	}
	if len(got) != 13 {
		t.Fatalf("len = %d, want 13", len(got))
	}
	if got[0].Directory != "/directory/1/" || got[0].Count != 5 || got[0].Bytes != 60141 {
		t.Errorf("got[0] = %+v", got[0])
	}
	if !got[0].HasSubDirs {
		t.Errorf("got[0].HasSubDirs = false, want true")
	}
	if got[3].HasSubDirs {
		t.Errorf("got[3].HasSubDirs = true, want false (fixture says N)")
	}
	// /directory/2/ is the largest entry — sanity-check that 9-digit byte
	// counts decode without overflow on 32-bit platforms.
	if got[1].Bytes != 356123958 {
		t.Errorf("got[1].Bytes = %d, want 356123958", got[1].Bytes)
	}
}

func TestDecodeTraffic(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "usage/get_traffic_response_success.xml")
	got, err := usage.DecodeTraffic(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeTraffic: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	summary := got[0]
	if summary.Day != 0 {
		t.Errorf("summary.Day = %d, want 0", summary.Day)
	}
	if summary.Year != 2026 || summary.Month != 5 {
		t.Errorf("summary year/month = %d/%d, want 2026/5", summary.Year, summary.Month)
	}
	if summary.HTTPTraffic != 4104052 || summary.HTTPHits != 216 {
		t.Errorf("summary http = %d bytes / %d hits", summary.HTTPTraffic, summary.HTTPHits)
	}
	// xsi:nil ftp_traffic + ftp_hits must round-trip as zero.
	if summary.FTPTraffic != 0 || summary.FTPHits != 0 {
		t.Errorf("summary ftp = %d/%d, want 0/0 (nil mapped to zero)",
			summary.FTPTraffic, summary.FTPHits)
	}
	if summary.Comment != "traffic summary for 2026-05" {
		t.Errorf("summary.Comment = %q", summary.Comment)
	}
	day := got[1]
	if day.Day != 1 {
		t.Errorf("day.Day = %d, want 1", day.Day)
	}
	if !summary.IsSummary() {
		t.Errorf("summary.IsSummary() = false, want true")
	}
	if day.IsSummary() {
		t.Errorf("day.IsSummary() = true, want false")
	}
}

func TestDecodeTrafficRejectsMalformedYear(t *testing.T) {
	t.Parallel()
	if _, err := usage.DecodeTraffic(trafficRow("abc", "05")); err == nil {
		t.Fatal("DecodeTraffic err = nil for non-numeric year, want a decode error")
	}
	if _, err := usage.DecodeTraffic(trafficRow("2026", "")); err == nil {
		t.Fatal("DecodeTraffic err = nil for empty month, want a decode error")
	}
	got, err := usage.DecodeTraffic(trafficRow("2026", "05"))
	if err != nil {
		t.Fatalf("DecodeTraffic on valid period: %v", err)
	}
	if len(got) != 1 || got[0].Year != 2026 || got[0].Month != 5 {
		t.Errorf("got = %+v, want one row with year 2026 month 5", got)
	}
}

func TestClientSpace(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "usage/get_space_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	list, err := usage.NewClient(fc).Space(context.Background())
	if err != nil {
		t.Fatalf("Space: %v", err)
	}
	if fc.GotAction != "get_space" {
		t.Errorf("action = %q, want get_space", fc.GotAction)
	}
	if fc.GotParams != nil {
		t.Errorf("params = %v, want nil", fc.GotParams)
	}
	if len(list) == 0 {
		t.Errorf("len = %d, want a non-empty list", len(list))
	}
}

func TestClientSpaceUsageWithDirectory(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "usage/get_space_usage_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	_, err := usage.NewClient(fc).SpaceUsage(context.Background(), "/htdocs")
	if err != nil {
		t.Fatalf("SpaceUsage: %v", err)
	}
	if fc.GotAction != "get_space_usage" {
		t.Errorf("action = %q, want get_space_usage", fc.GotAction)
	}
	if dir, ok := fc.GotParams["directory"].(string); !ok || dir != "/htdocs" {
		t.Errorf("params[directory] = %v, want /htdocs", fc.GotParams["directory"])
	}
}

func TestClientSpaceUsageEmptyDirectoryOmitsParams(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "usage/get_space_usage_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	if _, err := usage.NewClient(fc).SpaceUsage(context.Background(), ""); err != nil {
		t.Fatalf("SpaceUsage: %v", err)
	}
	if fc.GotParams != nil {
		t.Errorf("params = %v, want nil for empty directory", fc.GotParams)
	}
}

func TestClientTrafficZeroOmitsParams(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "usage/get_traffic_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	if _, err := usage.NewClient(fc).Traffic(context.Background(), 0, 0); err != nil {
		t.Fatalf("Traffic: %v", err)
	}
	if fc.GotParams != nil {
		t.Errorf("params = %v, want nil when year and month are zero", fc.GotParams)
	}
}

func TestClientTrafficWithYearMonthZeroPads(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "usage/get_traffic_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	if _, err := usage.NewClient(fc).Traffic(context.Background(), 2026, 3); err != nil {
		t.Fatalf("Traffic: %v", err)
	}
	if fc.GotParams["year"] != "2026" {
		t.Errorf("params[year] = %v, want \"2026\"", fc.GotParams["year"])
	}
	if fc.GotParams["month"] != "03" {
		t.Errorf("params[month] = %v, want \"03\" (zero-padded)", fc.GotParams["month"])
	}
}

func TestClientPropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := usage.NewClient(&testutil.FakeCaller{Err: want})
	if _, err := c.Space(context.Background()); !errors.Is(err, want) {
		t.Errorf("Space err = %v, want %v wrapped", err, want)
	}
	if _, err := c.SpaceUsage(context.Background(), ""); !errors.Is(err, want) {
		t.Errorf("SpaceUsage err = %v, want %v wrapped", err, want)
	}
	if _, err := c.Traffic(context.Background(), 0, 0); !errors.Is(err, want) {
		t.Errorf("Traffic err = %v, want %v wrapped", err, want)
	}
}

func TestSpaceListTabular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "usage/get_space_response_success.xml")
	list, _ := usage.DecodeSpace(resp.Body.ReturnInfo)
	rows := list.TableRows()
	if len(rows) != 5 {
		t.Fatalf("rows = %d, want 5", len(rows))
	}
	if rows[0][0] != "w0000001" || rows[0][1] != "11947780" || rows[0][2] != "36454400" {
		t.Errorf("rows[0] = %v", rows[0])
	}
}

func TestSpaceUsageListTabular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "usage/get_space_usage_response_success.xml")
	list, _ := usage.DecodeSpaceUsage(resp.Body.ReturnInfo)
	rows := list.TableRows()
	if len(rows) != 13 {
		t.Fatalf("rows = %d, want 13", len(rows))
	}
	if rows[0][0] != "/directory/1/" || rows[0][3] != "Y" {
		t.Errorf("rows[0] = %v", rows[0])
	}
	if rows[3][3] != "N" {
		t.Errorf("rows[3] = %v, want N in subdir column", rows[3])
	}
}

func TestTrafficListTabular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "usage/get_traffic_response_success.xml")
	list, _ := usage.DecodeTraffic(resp.Body.ReturnInfo)
	rows := list.TableRows()
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	if rows[0][3] != "*" {
		t.Errorf("summary row DAY column = %q, want *", rows[0][3])
	}
	if rows[1][3] != "01" {
		t.Errorf("day row DAY column = %q, want 01", rows[1][3])
	}
}
