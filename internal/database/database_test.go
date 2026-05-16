package database_test

import (
	"context"
	"errors"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/database"
	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestDecodeDatabases(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "database/get_databases_response_success.xml")
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
	resp := testutil.DecodeFixture(t, "database/get_database_response_success.xml")
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
	resp := testutil.DecodeFixture(t, "database/get_databases_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	list, err := database.NewClient(fc).List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if fc.GotAction != "get_databases" {
		t.Errorf("action = %q, want get_databases", fc.GotAction)
	}
	if fc.GotParams != nil {
		t.Errorf("params = %v, want nil", fc.GotParams)
	}
	if len(list) == 0 {
		t.Errorf("len = %d, want a non-empty list", len(list))
	}
}

func TestClientGet(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "database/get_database_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	d, err := database.NewClient(fc).Get(context.Background(), "d0123452")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if fc.GotAction != "get_databases" {
		t.Errorf("action = %q, want get_databases", fc.GotAction)
	}
	if got, _ := fc.GotParams["database_login"].(string); got != "d0123452" {
		t.Errorf("params[database_login] = %v, want d0123452", fc.GotParams["database_login"])
	}
	if d.Login != "d0123452" {
		t.Errorf("Login = %q, want d0123452", d.Login)
	}
}

func TestClientGetEmptyLogin(t *testing.T) {
	t.Parallel()
	c := database.NewClient(&testutil.FakeCaller{})
	if _, err := c.Get(context.Background(), ""); err == nil {
		t.Errorf("Get(\"\") err = nil, want validation error")
	}
}

func TestClientGetNotFound(t *testing.T) {
	t.Parallel()
	resp := &soap.Response{Body: soap.ResponseBody{ReturnInfo: soap.Value{Kind: soap.KindArray}}}
	c := database.NewClient(&testutil.FakeCaller{Resp: resp})
	if _, err := c.Get(context.Background(), "missing"); err == nil {
		t.Errorf("Get on empty result err = nil, want not-found")
	}
}

func TestClientPropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := database.NewClient(&testutil.FakeCaller{Err: want})
	if _, err := c.List(context.Background()); !errors.Is(err, want) {
		t.Errorf("List err = %v, want %v wrapped", err, want)
	}
	if _, err := c.Get(context.Background(), "d0123452"); !errors.Is(err, want) {
		t.Errorf("Get err = %v, want %v wrapped", err, want)
	}
}

func TestDatabaseListTabular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "database/get_databases_response_success.xml")
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
	resp := testutil.DecodeFixture(t, "database/get_database_response_success.xml")
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
