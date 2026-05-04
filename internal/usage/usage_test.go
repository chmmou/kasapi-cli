package usage_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/usage"
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
	path := filepath.Join(repoRoot(t), "testdata", "statistic", name)
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

func TestDecodeSpace(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_space_response_success.xml")
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
	resp := decodeFixture(t, "get_space_usage_response_success.xml")
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
	resp := decodeFixture(t, "get_traffic_response_success.xml")
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
}

func TestClientSpace(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_space_response_success.xml")
	fc := &fakeCaller{resp: resp}
	list, err := usage.NewClient(fc).Space(context.Background())
	if err != nil {
		t.Fatalf("Space: %v", err)
	}
	if fc.gotAction != "get_space" {
		t.Errorf("action = %q, want get_space", fc.gotAction)
	}
	if fc.gotParams != nil {
		t.Errorf("params = %v, want nil", fc.gotParams)
	}
	if len(list) != 5 {
		t.Errorf("len = %d, want 5", len(list))
	}
}

func TestClientSpaceUsageWithDirectory(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_space_usage_response_success.xml")
	fc := &fakeCaller{resp: resp}
	_, err := usage.NewClient(fc).SpaceUsage(context.Background(), "/htdocs")
	if err != nil {
		t.Fatalf("SpaceUsage: %v", err)
	}
	if fc.gotAction != "get_space_usage" {
		t.Errorf("action = %q, want get_space_usage", fc.gotAction)
	}
	if dir, ok := fc.gotParams["directory"].(string); !ok || dir != "/htdocs" {
		t.Errorf("params[directory] = %v, want /htdocs", fc.gotParams["directory"])
	}
}

func TestClientSpaceUsageEmptyDirectoryOmitsParams(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_space_usage_response_success.xml")
	fc := &fakeCaller{resp: resp}
	if _, err := usage.NewClient(fc).SpaceUsage(context.Background(), ""); err != nil {
		t.Fatalf("SpaceUsage: %v", err)
	}
	if fc.gotParams != nil {
		t.Errorf("params = %v, want nil for empty directory", fc.gotParams)
	}
}

func TestClientTrafficZeroOmitsParams(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_traffic_response_success.xml")
	fc := &fakeCaller{resp: resp}
	if _, err := usage.NewClient(fc).Traffic(context.Background(), 0, 0); err != nil {
		t.Fatalf("Traffic: %v", err)
	}
	if fc.gotParams != nil {
		t.Errorf("params = %v, want nil when year and month are zero", fc.gotParams)
	}
}

func TestClientTrafficWithYearMonthZeroPads(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_traffic_response_success.xml")
	fc := &fakeCaller{resp: resp}
	if _, err := usage.NewClient(fc).Traffic(context.Background(), 2026, 3); err != nil {
		t.Fatalf("Traffic: %v", err)
	}
	if fc.gotParams["year"] != "2026" {
		t.Errorf("params[year] = %v, want \"2026\"", fc.gotParams["year"])
	}
	if fc.gotParams["month"] != "03" {
		t.Errorf("params[month] = %v, want \"03\" (zero-padded)", fc.gotParams["month"])
	}
}

func TestClientPropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := usage.NewClient(&fakeCaller{err: want})
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
	resp := decodeFixture(t, "get_space_response_success.xml")
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
	resp := decodeFixture(t, "get_space_usage_response_success.xml")
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
	resp := decodeFixture(t, "get_traffic_response_success.xml")
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
