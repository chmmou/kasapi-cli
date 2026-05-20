package database_test

import (
	"context"
	"errors"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/database"
	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func sampleSpec() database.Spec {
	return database.Spec{
		Password:     "s3cret",
		Comment:      "Test Database for CLI Test",
		AllowedHosts: "localhost, 192.168.100.10, 192.168.100.10/32",
	}
}

func TestClientAdd(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "database/add_database_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	login, err := database.NewClient(fc).Add(context.Background(), sampleSpec())
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if fc.GotAction != "add_database" {
		t.Errorf("action = %q, want add_database", fc.GotAction)
	}
	wantParams := map[string]any{
		"database_password":      "s3cret",
		"database_comment":       "Test Database for CLI Test",
		"database_allowed_hosts": "localhost, 192.168.100.10, 192.168.100.10/32",
	}
	for k, v := range wantParams {
		if fc.GotParams[k] != v {
			t.Errorf("params[%q] = %v, want %v (full: %v)", k, fc.GotParams[k], v, fc.GotParams)
		}
	}
	// add_database must not carry a database_login (the server generates
	// it) nor the update-only database_new_password key.
	for _, k := range []string{"database_login", "database_new_password"} {
		if _, ok := fc.GotParams[k]; ok {
			t.Errorf("add_database must not send %q: %v", k, fc.GotParams)
		}
	}
	if login != "d0123460" {
		t.Errorf("returned login = %q, want d0123460 (fixture ReturnInfo)", login)
	}
}

func TestClientUpdate(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "database/update_database_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	fields := map[string]string{
		database.FieldNewPassword:  "n3wpass",
		database.FieldComment:      "Test Database for CLI Test Update",
		database.FieldAllowedHosts: "localhost, 192.168.100.10, 192.168.100.10/24",
	}
	if err := database.NewClient(fc).Update(context.Background(), "d0123460", fields); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if fc.GotAction != "update_database" {
		t.Errorf("action = %q, want update_database", fc.GotAction)
	}
	wantParams := map[string]any{
		"database_login":         "d0123460",
		"database_new_password":  "n3wpass",
		"database_comment":       "Test Database for CLI Test Update",
		"database_allowed_hosts": "localhost, 192.168.100.10, 192.168.100.10/24",
	}
	for k, v := range wantParams {
		if fc.GotParams[k] != v {
			t.Errorf("params[%q] = %v, want %v (full: %v)", k, fc.GotParams[k], v, fc.GotParams)
		}
	}
	// update_database uses database_new_password — the add-only
	// database_password key must not leak through.
	if _, ok := fc.GotParams["database_password"]; ok {
		t.Errorf("update_database must not send database_password (it uses database_new_password): %v", fc.GotParams)
	}
}

func TestClientDelete(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "database/delete_database_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	if err := database.NewClient(fc).Delete(context.Background(), "d0123456"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if fc.GotAction != "delete_database" {
		t.Errorf("action = %q, want delete_database", fc.GotAction)
	}
	if fc.GotParams["database_login"] != "d0123456" {
		t.Errorf("params = %v", fc.GotParams)
	}
}

func TestWriteValidation(t *testing.T) {
	t.Parallel()
	c := database.NewClient(&testutil.FakeCaller{})
	ctx := context.Background()

	for _, tc := range []struct {
		name string
		mut  func(*database.Spec)
	}{
		{"missing password", func(s *database.Spec) { s.Password = "" }},
		{"missing comment", func(s *database.Spec) { s.Comment = "" }},
		{"missing allowed_hosts", func(s *database.Spec) { s.AllowedHosts = "" }},
	} {
		s := sampleSpec()
		tc.mut(&s)
		if _, err := c.Add(ctx, s); err == nil {
			t.Errorf("Add %s: err = nil, want validation error", tc.name)
		}
	}
	if err := c.Update(ctx, "", map[string]string{database.FieldComment: "x"}); err == nil {
		t.Error("Update empty login: err = nil, want validation error")
	}
	if err := c.Update(ctx, "d0123460", nil); err == nil {
		t.Error("Update no fields: err = nil, want validation error")
	}
	if err := c.Delete(ctx, ""); err == nil {
		t.Error("Delete empty login: err = nil, want validation error")
	}
}

func TestUnexpectedReturnString(t *testing.T) {
	t.Parallel()
	resp := &soap.Response{Body: soap.ResponseBody{ReturnString: "FALSE"}}
	c := database.NewClient(&testutil.FakeCaller{Resp: resp})
	ctx := context.Background()
	if _, err := c.Add(ctx, sampleSpec()); !errors.Is(err, database.ErrUnexpectedReturnString) {
		t.Errorf("Add err = %v, want ErrUnexpectedReturnString", err)
	}
	if err := c.Update(ctx, "d0123460", map[string]string{database.FieldComment: "x"}); !errors.Is(err, database.ErrUnexpectedReturnString) {
		t.Errorf("Update err = %v, want ErrUnexpectedReturnString", err)
	}
	if err := c.Delete(ctx, "d0123460"); !errors.Is(err, database.ErrUnexpectedReturnString) {
		t.Errorf("Delete err = %v, want ErrUnexpectedReturnString", err)
	}
}

func TestWritePropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := database.NewClient(&testutil.FakeCaller{Err: want})
	ctx := context.Background()
	if _, err := c.Add(ctx, sampleSpec()); !errors.Is(err, want) {
		t.Errorf("Add err = %v, want %v", err, want)
	}
	if err := c.Update(ctx, "d0123460", map[string]string{database.FieldComment: "x"}); !errors.Is(err, want) {
		t.Errorf("Update err = %v, want %v", err, want)
	}
	if err := c.Delete(ctx, "d0123460"); !errors.Is(err, want) {
		t.Errorf("Delete err = %v, want %v", err, want)
	}
}

func TestParamBuilders(t *testing.T) {
	t.Parallel()
	add := database.AddParams(sampleSpec())
	wantAdd := map[string]any{
		"database_password":      "s3cret",
		"database_comment":       "Test Database for CLI Test",
		"database_allowed_hosts": "localhost, 192.168.100.10, 192.168.100.10/32",
	}
	for k, v := range wantAdd {
		if add[k] != v {
			t.Errorf("AddParams[%q] = %v, want %v", k, add[k], v)
		}
	}
	if len(add) != 3 {
		t.Errorf("AddParams has %d keys, want 3 (password/comment/allowed_hosts, no database_login)", len(add))
	}
	if _, ok := add["database_login"]; ok {
		t.Errorf("AddParams must not contain database_login: %v", add)
	}
	upd := database.UpdateParams("d0123460", map[string]string{
		database.FieldNewPassword: "n3wpass",
		database.FieldComment:     "c",
	})
	if upd["database_login"] != "d0123460" || upd["database_new_password"] != "n3wpass" || upd["database_comment"] != "c" {
		t.Errorf("UpdateParams = %v", upd)
	}
	if _, ok := upd["database_password"]; ok {
		t.Errorf("UpdateParams must not contain database_password (update uses _new_password): %v", upd)
	}
	del := database.DeleteParams("d0123460")
	if len(del) != 1 || del["database_login"] != "d0123460" {
		t.Errorf("DeleteParams = %v", del)
	}
}
