package subdomain_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/subdomain"
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
	path := filepath.Join(repoRoot(t), "testdata", "subdomain", name)
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

func TestDecodeSubdomains(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_subdomains_response_success.xml")
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
	resp := decodeFixture(t, "get_subdomains_response_success.xml")
	fc := &fakeCaller{resp: resp}
	list, err := subdomain.NewClient(fc).List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if fc.gotAction != "get_subdomains" {
		t.Errorf("action = %q, want get_subdomains", fc.gotAction)
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
	c := subdomain.NewClient(&fakeCaller{err: want})
	if _, err := c.List(context.Background()); !errors.Is(err, want) {
		t.Errorf("List err = %v, want %v wrapped", err, want)
	}
}

func TestSubdomainListTabular(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_subdomains_response_success.xml")
	list, _ := subdomain.DecodeSubdomains(resp.Body.ReturnInfo)
	rows := list.TableRows()
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	if rows[0][0] != "sub1.example.com" || rows[0][1] != "w0000000" {
		t.Errorf("rows[0] = %v", rows[0])
	}
}
