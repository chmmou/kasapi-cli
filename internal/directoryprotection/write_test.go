package directoryprotection_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/directoryprotection"
	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func sampleSpec() directoryprotection.Spec {
	return directoryprotection.Spec{
		User:     "protected_user",
		Path:     "/protected/directory/",
		Password: "s3cret",
		AuthName: "Protected Area",
	}
}

func TestClientAdd(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "directoryprotection/add_directoryprotection_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	user, err := directoryprotection.NewClient(fc).Add(context.Background(), sampleSpec())
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if fc.GotAction != "add_directoryprotection" {
		t.Errorf("action = %q, want add_directoryprotection", fc.GotAction)
	}
	wantParams := map[string]any{
		"directory_user":     "protected_user",
		"directory_path":     "/protected/directory/",
		"directory_password": "s3cret",
		"directory_authname": "Protected Area",
	}
	for k, v := range wantParams {
		if fc.GotParams[k] != v {
			t.Errorf("params[%q] = %v, want %v (full: %v)", k, fc.GotParams[k], v, fc.GotParams)
		}
	}
	if user != "protected_user" {
		t.Errorf("returned user = %q, want protected_user (fixture ReturnInfo)", user)
	}
}

func TestClientUpdate(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "directoryprotection/update_directoryprotection_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	fields := map[string]string{
		directoryprotection.FieldPassword: "n3wpass",
		directoryprotection.FieldAuthName: "Protected Area Updated",
	}
	if err := directoryprotection.NewClient(fc).Update(context.Background(), "/protected/directory/", "protected_user", fields); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if fc.GotAction != "update_directoryprotection" {
		t.Errorf("action = %q, want update_directoryprotection", fc.GotAction)
	}
	wantParams := map[string]any{
		"directory_path":     "/protected/directory/",
		"directory_user":     "protected_user",
		"directory_password": "n3wpass",
		"directory_authname": "Protected Area Updated",
	}
	for k, v := range wantParams {
		if fc.GotParams[k] != v {
			t.Errorf("params[%q] = %v, want %v (full: %v)", k, fc.GotParams[k], v, fc.GotParams)
		}
	}
}

// update must send only the explicitly-changed fields (keyed on cobra
// Changed at the CLI layer): updating the authname alone must not leak
// a directory_password key, so an omitted password keeps the current
// one instead of being reset.
func TestClientUpdateSparse(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "directoryprotection/update_directoryprotection_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	fields := map[string]string{directoryprotection.FieldAuthName: "Realm Only"}
	if err := directoryprotection.NewClient(fc).Update(context.Background(), "/protected/directory/", "protected_user", fields); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if _, ok := fc.GotParams["directory_password"]; ok {
		t.Errorf("authname-only update must not send directory_password: %v", fc.GotParams)
	}
	if fc.GotParams["directory_authname"] != "Realm Only" {
		t.Errorf("params[directory_authname] = %v, want Realm Only", fc.GotParams["directory_authname"])
	}
}

func TestClientDelete(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "directoryprotection/delete_directoryprotection_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	if err := directoryprotection.NewClient(fc).Delete(context.Background(), "/protected/directory/", "protected_user"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if fc.GotAction != "delete_directoryprotection" {
		t.Errorf("action = %q, want delete_directoryprotection", fc.GotAction)
	}
	if fc.GotParams["directory_path"] != "/protected/directory/" || fc.GotParams["directory_user"] != "protected_user" {
		t.Errorf("params = %v, want only path+user identity", fc.GotParams)
	}
	// delete carries the identity only — no password / authname.
	for _, k := range []string{"directory_password", "directory_authname"} {
		if _, ok := fc.GotParams[k]; ok {
			t.Errorf("delete_directoryprotection must not send %q: %v", k, fc.GotParams)
		}
	}
}

func TestWriteValidation(t *testing.T) {
	t.Parallel()
	c := directoryprotection.NewClient(&testutil.FakeCaller{})
	ctx := context.Background()

	// Each missing-field case must surface a per-field validation error
	// mentioning only that single field, not a combined message.
	for _, tc := range []struct {
		name    string
		mut     func(*directoryprotection.Spec)
		wantSub string
	}{
		{"missing user", func(s *directoryprotection.Spec) { s.User = "" }, "user"},
		{"missing path", func(s *directoryprotection.Spec) { s.Path = "" }, "path"},
		{"missing password", func(s *directoryprotection.Spec) { s.Password = "" }, "password"},
	} {
		s := sampleSpec()
		tc.mut(&s)
		_, err := c.Add(ctx, s)
		if err == nil {
			t.Errorf("Add %s: err = nil, want validation error", tc.name)
			continue
		}
		if !strings.Contains(err.Error(), tc.wantSub) {
			t.Errorf("Add %s: err = %q, want it to mention %q", tc.name, err.Error(), tc.wantSub)
		}
	}
	// AuthName == "" is a deliberate "no realm label" value, not a
	// missing parameter — Add must reach the SOAP call (intercepted by
	// the FakeCaller with a success fixture).
	resp := testutil.DecodeFixture(t, "directoryprotection/add_directoryprotection_response_success.xml")
	emptyRealm := directoryprotection.NewClient(&testutil.FakeCaller{Resp: resp})
	s := sampleSpec()
	s.AuthName = ""
	if _, err := emptyRealm.Add(ctx, s); err != nil {
		t.Errorf("Add with empty AuthName: err = %v, want nil (empty realm label is allowed)", err)
	}

	if err := c.Update(ctx, "", "u", map[string]string{directoryprotection.FieldAuthName: "x"}); err == nil {
		t.Error("Update empty path: err = nil, want validation error")
	}
	if err := c.Update(ctx, "/p/", "", map[string]string{directoryprotection.FieldAuthName: "x"}); err == nil {
		t.Error("Update empty user: err = nil, want validation error")
	}
	if err := c.Update(ctx, "/p/", "u", nil); err == nil {
		t.Error("Update no fields: err = nil, want validation error")
	}
	if err := c.Delete(ctx, "", "u"); err == nil {
		t.Error("Delete empty path: err = nil, want validation error")
	}
	if err := c.Delete(ctx, "/p/", ""); err == nil {
		t.Error("Delete empty user: err = nil, want validation error")
	}
}

func TestUnexpectedReturnString(t *testing.T) {
	t.Parallel()
	resp := &soap.Response{Body: soap.ResponseBody{ReturnString: "FALSE"}}
	c := directoryprotection.NewClient(&testutil.FakeCaller{Resp: resp})
	ctx := context.Background()
	if _, err := c.Add(ctx, sampleSpec()); !errors.Is(err, directoryprotection.ErrUnexpectedReturnString) {
		t.Errorf("Add err = %v, want ErrUnexpectedReturnString", err)
	}
	if err := c.Update(ctx, "/p/", "u", map[string]string{directoryprotection.FieldAuthName: "x"}); !errors.Is(err, directoryprotection.ErrUnexpectedReturnString) {
		t.Errorf("Update err = %v, want ErrUnexpectedReturnString", err)
	}
	if err := c.Delete(ctx, "/p/", "u"); !errors.Is(err, directoryprotection.ErrUnexpectedReturnString) {
		t.Errorf("Delete err = %v, want ErrUnexpectedReturnString", err)
	}
}

func TestWritePropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := directoryprotection.NewClient(&testutil.FakeCaller{Err: want})
	ctx := context.Background()
	if _, err := c.Add(ctx, sampleSpec()); !errors.Is(err, want) {
		t.Errorf("Add err = %v, want %v", err, want)
	}
	if err := c.Update(ctx, "/p/", "u", map[string]string{directoryprotection.FieldAuthName: "x"}); !errors.Is(err, want) {
		t.Errorf("Update err = %v, want %v", err, want)
	}
	if err := c.Delete(ctx, "/p/", "u"); !errors.Is(err, want) {
		t.Errorf("Delete err = %v, want %v", err, want)
	}
}

func TestParamBuilders(t *testing.T) {
	t.Parallel()
	add := directoryprotection.AddParams(sampleSpec())
	wantAdd := map[string]any{
		"directory_user":     "protected_user",
		"directory_path":     "/protected/directory/",
		"directory_password": "s3cret",
		"directory_authname": "Protected Area",
	}
	for k, v := range wantAdd {
		if add[k] != v {
			t.Errorf("AddParams[%q] = %v, want %v", k, add[k], v)
		}
	}
	if len(add) != 4 {
		t.Errorf("AddParams has %d keys, want 4 (user/path/password/authname)", len(add))
	}

	upd := directoryprotection.UpdateParams("/protected/directory/", "protected_user", map[string]string{
		directoryprotection.FieldPassword: "n3wpass",
	})
	if upd["directory_path"] != "/protected/directory/" || upd["directory_user"] != "protected_user" || upd["directory_password"] != "n3wpass" {
		t.Errorf("UpdateParams = %v", upd)
	}
	if _, ok := upd["directory_authname"]; ok {
		t.Errorf("UpdateParams must not contain unset directory_authname: %v", upd)
	}

	del := directoryprotection.DeleteParams("/protected/directory/", "protected_user")
	if len(del) != 2 || del["directory_path"] != "/protected/directory/" || del["directory_user"] != "protected_user" {
		t.Errorf("DeleteParams = %v", del)
	}
}
