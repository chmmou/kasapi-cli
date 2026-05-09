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

func TestDecodeMailingListSingular(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_mailinglist_response_success.xml")
	got, err := mailinglist.DecodeMailingLists(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeMailingLists: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Name != "announce@example.com" {
		t.Errorf("Name = %q", got[0].Name)
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

func TestClientGet(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_mailinglist_response_success.xml")
	fc := &fakeCaller{resp: resp}
	m, err := mailinglist.NewClient(fc).Get(context.Background(), "announce@example.com")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if fc.gotAction != "get_mailinglists" {
		t.Errorf("action = %q, want get_mailinglists", fc.gotAction)
	}
	if got, _ := fc.gotParams["mailinglist_name"].(string); got != "announce@example.com" {
		t.Errorf("params[mailinglist_name] = %v, want announce@example.com", fc.gotParams["mailinglist_name"])
	}
	if m.Name != "announce@example.com" {
		t.Errorf("Name = %q", m.Name)
	}
}

func TestClientGetEmptyName(t *testing.T) {
	t.Parallel()
	c := mailinglist.NewClient(&fakeCaller{})
	if _, err := c.Get(context.Background(), ""); err == nil {
		t.Errorf("Get(\"\") err = nil, want validation error")
	}
}

func TestClientGetNotFound(t *testing.T) {
	t.Parallel()
	resp := &soap.Response{Body: soap.ResponseBody{ReturnInfo: soap.Value{Kind: soap.KindArray}}}
	c := mailinglist.NewClient(&fakeCaller{resp: resp})
	if _, err := c.Get(context.Background(), "missing@example.com"); err == nil {
		t.Errorf("Get on empty result err = nil, want not-found")
	}
}

func TestClientPropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := mailinglist.NewClient(&fakeCaller{err: want})
	if _, err := c.List(context.Background()); !errors.Is(err, want) {
		t.Errorf("List err = %v, want %v wrapped", err, want)
	}
	if _, err := c.Get(context.Background(), "announce@example.com"); !errors.Is(err, want) {
		t.Errorf("Get err = %v, want %v wrapped", err, want)
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

func TestMailingListSingularTabular(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_mailinglist_response_success.xml")
	list, _ := mailinglist.DecodeMailingLists(resp.Body.ReturnInfo)
	if len(list) != 1 {
		t.Fatalf("len = %d, want 1", len(list))
	}
	rows := list[0].TableRows()
	if len(rows) != 4 {
		t.Fatalf("rows = %d, want 4", len(rows))
	}
	wantPairs := map[string]string{
		"mailinglist_name":  "announce@example.com",
		"mailinglist_admin": "admin@example.com",
		"mailinglist_url":   "https://lists.example.com/mailman/listinfo/announce",
		"in_progress":       "FALSE",
	}
	for _, r := range rows {
		if want, ok := wantPairs[r[0]]; !ok || r[1] != want {
			t.Errorf("row %v not in expected map", r)
		}
	}
	hdr := (mailinglist.MailingList{}).TableHeaders()
	if len(hdr) != 2 || hdr[0] != "FIELD" || hdr[1] != "VALUE" {
		t.Errorf("headers = %v, want [FIELD VALUE]", hdr)
	}
}
