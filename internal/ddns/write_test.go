package ddns_test

import (
	"context"
	"errors"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/ddns"
	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func sampleSpec() ddns.Spec {
	return ddns.Spec{
		Password:  "s3cret",
		Zone:      "example.com",
		Label:     "home",
		TargetIP:  "203.0.113.42",
		Comment:   "Home router",
		DualStack: "Y",
	}
}

func TestClientAdd(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "ddns/add_ddnsuser_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	login, err := ddns.NewClient(fc).Add(context.Background(), sampleSpec())
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if fc.GotAction != "add_ddnsuser" {
		t.Errorf("action = %q, want add_ddnsuser", fc.GotAction)
	}
	wantParams := map[string]any{
		"dyndns_password":   "s3cret",
		"dyndns_zone":       "example.com",
		"dyndns_label":      "home",
		"dyndns_target_ip":  "203.0.113.42",
		"dyndns_comment":    "Home router",
		"dyndns_dual_stack": "Y",
	}
	for k, v := range wantParams {
		if fc.GotParams[k] != v {
			t.Errorf("params[%q] = %v, want %v (full: %v)", k, fc.GotParams[k], v, fc.GotParams)
		}
	}
	// add_ddnsuser must not carry a dyndns_login (the server generates
	// it) and must not use the update-only ipv4/ipv6 keys.
	for _, k := range []string{"dyndns_login", "dyndns_target_ipv4", "dyndns_target_ipv6"} {
		if _, ok := fc.GotParams[k]; ok {
			t.Errorf("add_ddnsuser must not send %q: %v", k, fc.GotParams)
		}
	}
	if login != "dyn0000001" {
		t.Errorf("returned login = %q, want dyn0000001 (fixture ReturnInfo)", login)
	}
}

func TestClientUpdate(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "ddns/update_ddnsuser_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	fields := map[string]string{
		ddns.FieldPassword:   "n3wpass",
		ddns.FieldComment:    "Home router (renamed)",
		ddns.FieldTargetIPv4: "127.0.0.1",
		ddns.FieldTargetIPv6: "::ffff:7f00:1",
		ddns.FieldDualStack:  "Y",
	}
	if err := ddns.NewClient(fc).Update(context.Background(), "dyn0000001", fields); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if fc.GotAction != "update_ddnsuser" {
		t.Errorf("action = %q, want update_ddnsuser", fc.GotAction)
	}
	wantParams := map[string]any{
		"dyndns_login":       "dyn0000001",
		"dyndns_password":    "n3wpass",
		"dyndns_comment":     "Home router (renamed)",
		"dyndns_target_ipv4": "127.0.0.1",
		"dyndns_target_ipv6": "::ffff:7f00:1",
		"dyndns_dual_stack":  "Y",
	}
	for k, v := range wantParams {
		if fc.GotParams[k] != v {
			t.Errorf("params[%q] = %v, want %v (full: %v)", k, fc.GotParams[k], v, fc.GotParams)
		}
	}
	// update_ddnsuser uses the same dyndns_password key as add (no
	// _new_password split) and must not leak the legacy single-IP key
	// into a dual-stack update.
	for _, k := range []string{"dyndns_new_password", "dyndns_target_ip"} {
		if _, ok := fc.GotParams[k]; ok {
			t.Errorf("update_ddnsuser must not send %q (fixture uses dyndns_password + ipv4/ipv6): %v", k, fc.GotParams)
		}
	}
}

func TestClientDelete(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "ddns/delete_ddnsuser_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	if err := ddns.NewClient(fc).Delete(context.Background(), "dyn0000001"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if fc.GotAction != "delete_ddnsuser" {
		t.Errorf("action = %q, want delete_ddnsuser", fc.GotAction)
	}
	if fc.GotParams["dyndns_login"] != "dyn0000001" {
		t.Errorf("params = %v", fc.GotParams)
	}
}

func TestWriteValidation(t *testing.T) {
	t.Parallel()
	c := ddns.NewClient(&testutil.FakeCaller{})
	ctx := context.Background()

	for _, tc := range []struct {
		name string
		mut  func(*ddns.Spec)
	}{
		{"missing password", func(s *ddns.Spec) { s.Password = "" }},
		{"missing zone", func(s *ddns.Spec) { s.Zone = "" }},
		{"missing label", func(s *ddns.Spec) { s.Label = "" }},
		{"missing target IP", func(s *ddns.Spec) { s.TargetIP = "" }},
		{"missing comment", func(s *ddns.Spec) { s.Comment = "" }},
	} {
		s := sampleSpec()
		tc.mut(&s)
		if _, err := c.Add(ctx, s); err == nil {
			t.Errorf("Add %s: err = nil, want validation error", tc.name)
		}
	}
	if err := c.Update(ctx, "", map[string]string{ddns.FieldComment: "x"}); err == nil {
		t.Error("Update empty login: err = nil, want validation error")
	}
	if err := c.Update(ctx, "dyn0000001", nil); err == nil {
		t.Error("Update no fields: err = nil, want validation error")
	}
	if err := c.Delete(ctx, ""); err == nil {
		t.Error("Delete empty login: err = nil, want validation error")
	}
}

func TestUnexpectedReturnString(t *testing.T) {
	t.Parallel()
	resp := &soap.Response{Body: soap.ResponseBody{ReturnString: "FALSE"}}
	c := ddns.NewClient(&testutil.FakeCaller{Resp: resp})
	ctx := context.Background()
	if _, err := c.Add(ctx, sampleSpec()); !errors.Is(err, ddns.ErrUnexpectedReturnString) {
		t.Errorf("Add err = %v, want ErrUnexpectedReturnString", err)
	}
	if err := c.Update(ctx, "dyn0000001", map[string]string{ddns.FieldComment: "x"}); !errors.Is(err, ddns.ErrUnexpectedReturnString) {
		t.Errorf("Update err = %v, want ErrUnexpectedReturnString", err)
	}
	if err := c.Delete(ctx, "dyn0000001"); !errors.Is(err, ddns.ErrUnexpectedReturnString) {
		t.Errorf("Delete err = %v, want ErrUnexpectedReturnString", err)
	}
}

func TestWritePropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := ddns.NewClient(&testutil.FakeCaller{Err: want})
	ctx := context.Background()
	if _, err := c.Add(ctx, sampleSpec()); !errors.Is(err, want) {
		t.Errorf("Add err = %v, want %v", err, want)
	}
	if err := c.Update(ctx, "dyn0000001", map[string]string{ddns.FieldComment: "x"}); !errors.Is(err, want) {
		t.Errorf("Update err = %v, want %v", err, want)
	}
	if err := c.Delete(ctx, "dyn0000001"); !errors.Is(err, want) {
		t.Errorf("Delete err = %v, want %v", err, want)
	}
}

func TestParamBuilders(t *testing.T) {
	t.Parallel()
	add := ddns.AddParams(sampleSpec())
	wantAdd := map[string]any{
		"dyndns_password":   "s3cret",
		"dyndns_zone":       "example.com",
		"dyndns_label":      "home",
		"dyndns_target_ip":  "203.0.113.42",
		"dyndns_comment":    "Home router",
		"dyndns_dual_stack": "Y",
	}
	for k, v := range wantAdd {
		if add[k] != v {
			t.Errorf("AddParams[%q] = %v, want %v", k, add[k], v)
		}
	}
	if len(add) != 6 {
		t.Errorf("AddParams has %d keys, want 6 (password/zone/label/target_ip/comment/dual_stack, no dyndns_login)", len(add))
	}
	if _, ok := add["dyndns_login"]; ok {
		t.Errorf("AddParams must not contain dyndns_login: %v", add)
	}
	upd := ddns.UpdateParams("dyn0000001", map[string]string{
		ddns.FieldComment:    "c",
		ddns.FieldTargetIPv4: "127.0.0.1",
	})
	if upd["dyndns_login"] != "dyn0000001" || upd["dyndns_comment"] != "c" || upd["dyndns_target_ipv4"] != "127.0.0.1" {
		t.Errorf("UpdateParams = %v", upd)
	}
	del := ddns.DeleteParams("dyn0000001")
	if len(del) != 1 || del["dyndns_login"] != "dyn0000001" {
		t.Errorf("DeleteParams = %v", del)
	}
}
