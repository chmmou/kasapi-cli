package server_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/server"
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

func decodeFixture(t *testing.T) *soap.Response {
	t.Helper()
	path := filepath.Join(repoRoot(t), "testdata", "account", "get_server_information_response_success.xml")
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = f.Close() }()
	resp, err := soap.Decode(f)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	return resp
}

func TestDecodeServices(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t)
	got, err := server.DecodeServices(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeServices: %v", err)
	}
	if len(got) != 8 {
		t.Fatalf("len = %d, want 8", len(got))
	}
	if got[0].Service != "mysql" || got[0].Version != "10.6.12" || got[0].VersionType != "server" {
		t.Errorf("got[0] = %+v", got[0])
	}
	if got[1].Service != "php" || got[1].FileExtension != "php56" || got[1].Interface != "cgi-fcgi" {
		t.Errorf("got[1] = %+v", got[1])
	}
	last := got[len(got)-1]
	if last.Service != "os" || last.Distribution != "ubuntu" || last.Version != "22" {
		t.Errorf("os entry = %+v", last)
	}
}

type fakeCaller struct {
	resp *soap.Response
	err  error
}

func (f fakeCaller) Call(_ context.Context, _ string, _ map[string]any) (*soap.Response, error) {
	return f.resp, f.err
}

func TestClientInformation(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t)
	c := server.NewClient(fakeCaller{resp: resp})
	list, err := c.Information(context.Background())
	if err != nil {
		t.Fatalf("Information: %v", err)
	}
	if len(list) != 8 {
		t.Errorf("len = %d, want 8", len(list))
	}
}

func TestClientInformationPropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := server.NewClient(fakeCaller{err: want})
	if _, err := c.Information(context.Background()); !errors.Is(err, want) {
		t.Errorf("err = %v, want %v wrapped", err, want)
	}
}

func TestServiceListTabular(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t)
	list, _ := server.DecodeServices(resp.Body.ReturnInfo)
	rows := list.TableRows()
	if len(rows) != 8 {
		t.Errorf("rows = %d, want 8", len(rows))
	}
	if rows[0][0] != "mysql" || rows[0][4] != "server" {
		t.Errorf("rows[0] = %v", rows[0])
	}
}
