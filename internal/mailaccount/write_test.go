package mailaccount_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/mailaccount"
	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func sampleSpec() mailaccount.Spec {
	return mailaccount.Spec{
		LocalPart:            "info",
		DomainPart:           "example.com",
		Password:             "s3cret",
		WebmailAutologin:     "Y",
		Responder:            "N",
		ResponderContentType: "text",
		ResponderDisplayName: "Info",
		XListEnabled:         "Y",
		XListSent:            "Sent",
		XListDrafts:          "Drafts",
		XListTrash:           "Trash",
		XListSpam:            "Spam",
		XListArchiv:          "Archive",
	}
}

func TestClientAdd(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "mailaccount/add_mailaccount_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	login, err := mailaccount.NewClient(fc).Add(context.Background(), sampleSpec())
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if fc.GotAction != "add_mailaccount" {
		t.Errorf("action = %q, want add_mailaccount", fc.GotAction)
	}
	wantParams := map[string]any{
		"local_part":                  "info",
		"domain_part":                 "example.com",
		"mail_password":               "s3cret",
		"webmail_autologin":           "Y",
		"responder":                   "N",
		"mail_responder_content_type": "text",
		"mail_responder_displayname":  "Info",
		"mail_xlist_enabled":          "Y",
		"mail_xlist_sent":             "Sent",
	}
	for k, v := range wantParams {
		if fc.GotParams[k] != v {
			t.Errorf("params[%q] = %v, want %v (full: %v)", k, fc.GotParams[k], v, fc.GotParams)
		}
	}
	// add_mailaccount must not carry a mail_login (the server generates
	// it), nor the update-only mail_new_password / is_active keys.
	for _, k := range []string{"mail_login", "mail_new_password", "is_active"} {
		if _, ok := fc.GotParams[k]; ok {
			t.Errorf("add_mailaccount must not send %q: %v", k, fc.GotParams)
		}
	}
	if login != "m0000001" {
		t.Errorf("returned login = %q, want m0000001 (fixture ReturnInfo)", login)
	}
}

func TestClientUpdate(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "mailaccount/update_mailaccount_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	fields := map[string]string{
		mailaccount.FieldNewPassword: "n3wpass",
		mailaccount.FieldIsActive:    "Y",
		mailaccount.FieldXListDrafts: "Entwürfe",
		mailaccount.FieldResponder:   "N",
	}
	if err := mailaccount.NewClient(fc).Update(context.Background(), "m0000001", fields); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if fc.GotAction != "update_mailaccount" {
		t.Errorf("action = %q, want update_mailaccount", fc.GotAction)
	}
	wantParams := map[string]any{
		"mail_login":        "m0000001",
		"mail_new_password": "n3wpass",
		"is_active":         "Y",
		"mail_xlist_drafts": "Entwürfe",
		"responder":         "N",
	}
	for k, v := range wantParams {
		if fc.GotParams[k] != v {
			t.Errorf("params[%q] = %v, want %v (full: %v)", k, fc.GotParams[k], v, fc.GotParams)
		}
	}
	// update_mailaccount uses mail_new_password — the add-only
	// mail_password key must not leak through.
	if _, ok := fc.GotParams["mail_password"]; ok {
		t.Errorf("update_mailaccount must not send mail_password (it uses mail_new_password): %v", fc.GotParams)
	}
}

func TestClientDelete(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "mailaccount/delete_mailaccount_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	if err := mailaccount.NewClient(fc).Delete(context.Background(), "m0000001"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if fc.GotAction != "delete_mailaccount" {
		t.Errorf("action = %q, want delete_mailaccount", fc.GotAction)
	}
	if fc.GotParams["mail_login"] != "m0000001" {
		t.Errorf("params = %v", fc.GotParams)
	}
}

func TestWriteValidation(t *testing.T) {
	t.Parallel()
	c := mailaccount.NewClient(&testutil.FakeCaller{})
	ctx := context.Background()

	// Each missing required add field surfaces a per-field validation
	// error (mentioning only that single field), not a combined message
	// — the latter forces the caller to guess which field broke.
	for _, tc := range []struct {
		name    string
		mut     func(*mailaccount.Spec)
		wantSub string
	}{
		{"missing local part", func(s *mailaccount.Spec) { s.LocalPart = "" }, "local part"},
		{"missing domain part", func(s *mailaccount.Spec) { s.DomainPart = "" }, "domain part"},
		{"missing password", func(s *mailaccount.Spec) { s.Password = "" }, "password"},
	} {
		s := sampleSpec()
		tc.mut(&s)
		if _, err := c.Add(ctx, s); err == nil {
			t.Errorf("Add %s: err = nil, want validation error", tc.name)
		} else if !strings.Contains(err.Error(), tc.wantSub) {
			t.Errorf("Add %s: err = %q, want it to mention %q", tc.name, err.Error(), tc.wantSub)
		}
	}
	if err := c.Update(ctx, "", map[string]string{mailaccount.FieldIsActive: "Y"}); err == nil {
		t.Error("Update empty login: err = nil, want validation error")
	}
	if err := c.Update(ctx, "m0000001", nil); err == nil {
		t.Error("Update no fields: err = nil, want validation error")
	}
	if err := c.Delete(ctx, ""); err == nil {
		t.Error("Delete empty login: err = nil, want validation error")
	}
}

func TestUnexpectedReturnString(t *testing.T) {
	t.Parallel()
	resp := &soap.Response{Body: soap.ResponseBody{ReturnString: "FALSE"}}
	c := mailaccount.NewClient(&testutil.FakeCaller{Resp: resp})
	ctx := context.Background()
	if _, err := c.Add(ctx, sampleSpec()); !errors.Is(err, mailaccount.ErrUnexpectedReturnString) {
		t.Errorf("Add err = %v, want ErrUnexpectedReturnString", err)
	}
	if err := c.Update(ctx, "m0000001", map[string]string{mailaccount.FieldIsActive: "Y"}); !errors.Is(err, mailaccount.ErrUnexpectedReturnString) {
		t.Errorf("Update err = %v, want ErrUnexpectedReturnString", err)
	}
	if err := c.Delete(ctx, "m0000001"); !errors.Is(err, mailaccount.ErrUnexpectedReturnString) {
		t.Errorf("Delete err = %v, want ErrUnexpectedReturnString", err)
	}
}

func TestWritePropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := mailaccount.NewClient(&testutil.FakeCaller{Err: want})
	ctx := context.Background()
	if _, err := c.Add(ctx, sampleSpec()); !errors.Is(err, want) {
		t.Errorf("Add err = %v, want %v", err, want)
	}
	if err := c.Update(ctx, "m0000001", map[string]string{mailaccount.FieldIsActive: "Y"}); !errors.Is(err, want) {
		t.Errorf("Update err = %v, want %v", err, want)
	}
	if err := c.Delete(ctx, "m0000001"); !errors.Is(err, want) {
		t.Errorf("Delete err = %v, want %v", err, want)
	}
}

func TestParamBuilders(t *testing.T) {
	t.Parallel()
	add := mailaccount.AddParams(sampleSpec())
	if add["local_part"] != "info" || add["domain_part"] != "example.com" || add["mail_password"] != "s3cret" {
		t.Errorf("AddParams identity fields = %v", add)
	}
	// add must not contain the update-only keys.
	for _, k := range []string{"mail_login", "mail_new_password", "is_active"} {
		if _, ok := add[k]; ok {
			t.Errorf("AddParams must not contain %q: %v", k, add)
		}
	}
	upd := mailaccount.UpdateParams("m0000001", map[string]string{
		mailaccount.FieldNewPassword: "n3wpass",
		mailaccount.FieldIsActive:    "N",
	})
	if upd["mail_login"] != "m0000001" || upd["mail_new_password"] != "n3wpass" || upd["is_active"] != "N" {
		t.Errorf("UpdateParams = %v", upd)
	}
	if _, ok := upd["mail_password"]; ok {
		t.Errorf("UpdateParams must not contain mail_password (update uses mail_new_password): %v", upd)
	}
	del := mailaccount.DeleteParams("m0000001")
	if len(del) != 1 || del["mail_login"] != "m0000001" {
		t.Errorf("DeleteParams = %v", del)
	}
}
