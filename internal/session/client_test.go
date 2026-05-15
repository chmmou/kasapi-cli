package session_test

import (
	"context"
	"errors"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/session"
	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestClientDeleteSuccess(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "session/delete_session_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	if err := session.NewClient(fc).Delete(context.Background()); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if fc.GotAction != "delete_session" {
		t.Errorf("action = %q, want delete_session", fc.GotAction)
	}
	if fc.GotParams != nil {
		t.Errorf("params = %v, want nil (delete_session takes no KasRequestParams)", fc.GotParams)
	}
}

func TestClientDeletePropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	fc := &testutil.FakeCaller{Err: want}
	if err := session.NewClient(fc).Delete(context.Background()); !errors.Is(err, want) {
		t.Errorf("Delete err = %v, want %v wrapped", err, want)
	}
}

func TestClientDeleteRejectsNilResponse(t *testing.T) {
	t.Parallel()
	fc := &testutil.FakeCaller{Resp: nil, Err: nil}
	if err := session.NewClient(fc).Delete(context.Background()); err == nil {
		t.Error("Delete err = nil, want error for nil response without error from Caller")
	}
}

func TestClientDeleteRejectsNonTrueReturnString(t *testing.T) {
	t.Parallel()
	resp := &soap.Response{Body: soap.ResponseBody{ReturnString: "FALSE"}}
	fc := &testutil.FakeCaller{Resp: resp}
	if err := session.NewClient(fc).Delete(context.Background()); err == nil {
		t.Error("Delete err = nil, want contract-violation error for ReturnString != TRUE")
	}
}
