package directoryprotection_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/directoryprotection"
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
	path := filepath.Join(repoRoot(t), "testdata", "directoryprotection", name)
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

func TestDecodeDirectoryProtections(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_directoryprotections_response_success.xml")
	got, err := directoryprotection.DecodeDirectoryProtections(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeDirectoryProtections: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	d := got[0]
	if d.User != "w0000000" {
		t.Errorf("User = %q", d.User)
	}
	if d.Path != "/protected/directory/" {
		t.Errorf("Path = %q", d.Path)
	}
	if d.AuthName != "ByPassword" {
		t.Errorf("AuthName = %q", d.AuthName)
	}
	if d.InProgress != "FALSE" {
		t.Errorf("InProgress = %q", d.InProgress)
	}
}

func TestDecodeDirectoryProtectionSingular(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_directoryprotection_response_success.xml")
	got, err := directoryprotection.DecodeDirectoryProtections(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeDirectoryProtections: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Path != "/protected/directory/" {
		t.Errorf("Path = %q", got[0].Path)
	}
}

func TestClientListNoFilter(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_directoryprotections_response_success.xml")
	fc := &fakeCaller{resp: resp}
	list, err := directoryprotection.NewClient(fc).List(context.Background(), "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if fc.gotAction != "get_directoryprotection" {
		t.Errorf("action = %q, want get_directoryprotection", fc.gotAction)
	}
	if fc.gotParams != nil {
		t.Errorf("params = %v, want nil", fc.gotParams)
	}
	if len(list) != 1 {
		t.Errorf("len = %d, want 1", len(list))
	}
}

func TestClientListWithPath(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_directoryprotection_response_success.xml")
	fc := &fakeCaller{resp: resp}
	list, err := directoryprotection.NewClient(fc).List(context.Background(), "/protected/directory/")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if fc.gotAction != "get_directoryprotection" {
		t.Errorf("action = %q", fc.gotAction)
	}
	if got, _ := fc.gotParams["directory_path"].(string); got != "/protected/directory/" {
		t.Errorf("params[directory_path] = %v", fc.gotParams["directory_path"])
	}
	if len(list) != 1 {
		t.Errorf("len = %d, want 1", len(list))
	}
}

func TestClientPropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := directoryprotection.NewClient(&fakeCaller{err: want})
	if _, err := c.List(context.Background(), ""); !errors.Is(err, want) {
		t.Errorf("List err = %v, want %v wrapped", err, want)
	}
	if _, err := c.List(context.Background(), "/foo/"); !errors.Is(err, want) {
		t.Errorf("List(path) err = %v, want %v wrapped", err, want)
	}
}

func TestDirectoryProtectionListTabular(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_directoryprotections_response_success.xml")
	list, _ := directoryprotection.DecodeDirectoryProtections(resp.Body.ReturnInfo)
	headers := list.TableHeaders()
	if headers[0] != "PATH" {
		t.Errorf("headers[0] = %q, want PATH", headers[0])
	}
	rows := list.TableRows()
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	if rows[0][0] != "/protected/directory/" {
		t.Errorf("rows[0][PATH] = %q", rows[0][0])
	}
}
