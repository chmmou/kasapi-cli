package mailinglist_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/mailinglist"
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
	path := filepath.Join(repoRoot(t), "testdata", "mailinglist", name)
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

func TestDecodeMailingLists(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_mailinglists_response_success.xml")
	got, err := mailinglist.DecodeMailingLists(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeMailingLists: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (per fixture arrayType)", len(got))
	}
	m := got[0]
	if m.Name != "announce@example.com" {
		t.Errorf("Name = %q", m.Name)
	}
	if m.Admin != "admin@example.com" {
		t.Errorf("Admin = %q", m.Admin)
	}
	if m.URL == "" {
		t.Errorf("URL empty")
	}
}

func TestClientList(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_mailinglists_response_success.xml")
	fc := &fakeCaller{resp: resp}
	list, err := mailinglist.NewClient(fc).List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if fc.gotAction != "get_mailinglists" {
		t.Errorf("action = %q, want get_mailinglists", fc.gotAction)
	}
	if fc.gotParams != nil {
		t.Errorf("params = %v, want nil", fc.gotParams)
	}
	if len(list) != 2 {
		t.Errorf("len = %d, want 2", len(list))
	}
}

func TestClientPropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := mailinglist.NewClient(&fakeCaller{err: want})
	if _, err := c.List(context.Background()); !errors.Is(err, want) {
		t.Errorf("List err = %v, want %v wrapped", err, want)
	}
}

func TestMailingListListTabular(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_mailinglists_response_success.xml")
	list, _ := mailinglist.DecodeMailingLists(resp.Body.ReturnInfo)
	rows := list.TableRows()
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	if rows[0][0] != "announce@example.com" {
		t.Errorf("rows[0][0] = %q", rows[0][0])
	}
}
