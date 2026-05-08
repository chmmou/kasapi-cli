package mailfilter_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/mailfilter"
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
	path := filepath.Join(repoRoot(t), "testdata", "mailfilter", name)
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

func TestDecodeStandardFilters(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_mailstandardfilter_response_success.xml")
	got, err := mailfilter.DecodeStandardFilters(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeStandardFilters: %v", err)
	}
	if len(got) != 9 {
		t.Fatalf("len = %d, want 9 (per fixture arrayType)", len(got))
	}
	if got[0].Filter != "rspamd" || got[0].Type != "rspamd" || got[0].Recommended != "Y" {
		t.Errorf("got[0] = %+v", got[0])
	}
	// Spot-check an entry whose type differs from the filter id.
	for _, f := range got {
		if f.Filter == "pdw" {
			if f.Type != "reject" || f.Title != "policyd-weight" {
				t.Errorf("pdw entry = %+v", f)
			}
			return
		}
	}
	t.Errorf("pdw entry missing")
}

func TestClientList(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_mailstandardfilter_response_success.xml")
	fc := &fakeCaller{resp: resp}
	list, err := mailfilter.NewClient(fc).List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if fc.gotAction != "get_mailstandardfilter" {
		t.Errorf("action = %q, want get_mailstandardfilter", fc.gotAction)
	}
	if fc.gotParams != nil {
		t.Errorf("params = %v, want nil", fc.gotParams)
	}
	if len(list) != 9 {
		t.Errorf("len = %d, want 9", len(list))
	}
}

func TestClientPropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := mailfilter.NewClient(&fakeCaller{err: want})
	if _, err := c.List(context.Background()); !errors.Is(err, want) {
		t.Errorf("List err = %v, want %v wrapped", err, want)
	}
}

func TestStandardFilterListTabular(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_mailstandardfilter_response_success.xml")
	list, _ := mailfilter.DecodeStandardFilters(resp.Body.ReturnInfo)
	rows := list.TableRows()
	if len(rows) != 9 {
		t.Fatalf("rows = %d, want 9", len(rows))
	}
	if rows[0][0] != "rspamd" {
		t.Errorf("rows[0][0] = %q, want rspamd", rows[0][0])
	}
}
