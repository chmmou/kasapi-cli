package database_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/database"
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
	path := filepath.Join(repoRoot(t), "testdata", "database", name)
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

func TestDecodeDatabases(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_databases_response_success.xml")
	got, err := database.DecodeDatabases(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeDatabases: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("len = %d, want 4", len(got))
	}
	d := got[0]
	if d.Login != "d0123450" {
		t.Errorf("Login = %q, want d0123450", d.Login)
	}
	if d.Name != "d0123450" {
		t.Errorf("Name = %q, want d0123450", d.Name)
	}
	if d.Comment != "my database comment" {
		t.Errorf("Comment = %q", d.Comment)
	}
	if d.UsedDatabaseSpace == 0 {
		t.Errorf("UsedDatabaseSpace = 0, want non-zero from xsd:float")
	}
	// d0123451 is the only entry with a non-empty allowed_hosts in the
	// fixture; verify the empty-string default survives for the others.
	if got[1].AllowedHosts != "localhost" {
		t.Errorf("got[1].AllowedHosts = %q, want localhost", got[1].AllowedHosts)
	}
	if got[0].AllowedHosts != "" {
		t.Errorf("got[0].AllowedHosts = %q, want empty", got[0].AllowedHosts)
	}
}

func TestDecodeDatabaseSingular(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_database_response_success.xml")
	got, err := database.DecodeDatabases(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeDatabases: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	d := got[0]
	if d.Login != "d0123452" {
		t.Errorf("Login = %q, want d0123452", d.Login)
	}
	if d.UsedDatabaseSpace == 0 {
		t.Errorf("UsedDatabaseSpace = 0, want non-zero")
	}
}

func TestClientList(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_databases_response_success.xml")
	fc := &fakeCaller{resp: resp}
	list, err := database.NewClient(fc).List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if fc.gotAction != "get_databases" {
		t.Errorf("action = %q, want get_databases", fc.gotAction)
	}
	if fc.gotParams != nil {
		t.Errorf("params = %v, want nil", fc.gotParams)
	}
	if len(list) != 4 {
		t.Errorf("len = %d, want 4", len(list))
	}
}

func TestClientGet(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_database_response_success.xml")
	fc := &fakeCaller{resp: resp}
	d, err := database.NewClient(fc).Get(context.Background(), "d0123452")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if fc.gotAction != "get_databases" {
		t.Errorf("action = %q, want get_databases", fc.gotAction)
	}
	if got, _ := fc.gotParams["database_login"].(string); got != "d0123452" {
		t.Errorf("params[database_login] = %v, want d0123452", fc.gotParams["database_login"])
	}
	if d.Login != "d0123452" {
		t.Errorf("Login = %q, want d0123452", d.Login)
	}
}

func TestClientGetEmptyLogin(t *testing.T) {
	t.Parallel()
	c := database.NewClient(&fakeCaller{})
	if _, err := c.Get(context.Background(), ""); err == nil {
		t.Errorf("Get(\"\") err = nil, want validation error")
	}
}

func TestClientGetNotFound(t *testing.T) {
	t.Parallel()
	resp := &soap.Response{Body: soap.ResponseBody{ReturnInfo: soap.Value{Kind: soap.KindArray}}}
	c := database.NewClient(&fakeCaller{resp: resp})
	if _, err := c.Get(context.Background(), "missing"); err == nil {
		t.Errorf("Get on empty result err = nil, want not-found")
	}
}

func TestClientPropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := database.NewClient(&fakeCaller{err: want})
	if _, err := c.List(context.Background()); !errors.Is(err, want) {
		t.Errorf("List err = %v, want %v wrapped", err, want)
	}
	if _, err := c.Get(context.Background(), "d0123452"); !errors.Is(err, want) {
		t.Errorf("Get err = %v, want %v wrapped", err, want)
	}
}

func TestDatabaseListTabular(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_databases_response_success.xml")
	list, _ := database.DecodeDatabases(resp.Body.ReturnInfo)
	headers := list.TableHeaders()
	if headers[0] != "LOGIN" {
		t.Errorf("headers[0] = %q, want LOGIN", headers[0])
	}
	rows := list.TableRows()
	if len(rows) != 4 {
		t.Fatalf("rows = %d, want 4", len(rows))
	}
	if rows[0][0] != "d0123450" {
		t.Errorf("rows[0][0] = %q, want d0123450", rows[0][0])
	}
}

func TestDatabaseTabular(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_database_response_success.xml")
	list, _ := database.DecodeDatabases(resp.Body.ReturnInfo)
	if len(list) != 1 {
		t.Fatalf("len = %d, want 1", len(list))
	}
	d := list[0]
	headers := d.TableHeaders()
	if headers[0] != "FIELD" || headers[1] != "VALUE" {
		t.Errorf("headers = %v, want [FIELD VALUE]", headers)
	}
	rows := d.TableRows()
	if rows[0][0] != "database_login" || rows[0][1] != "d0123452" {
		t.Errorf("rows[0] = %v, want [database_login d0123452]", rows[0])
	}
}
