package kaswrite_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/kaswrite"
	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestCallSuccessPassthrough(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "mailforward/add_mailforward_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	got, err := kaswrite.Call(context.Background(), fc, "mailforward", "add_mailforward", nil)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if got != resp {
		t.Errorf("Call resp = %p, want passthrough of %p", got, resp)
	}
	if fc.GotAction != "add_mailforward" {
		t.Errorf("action = %q, want add_mailforward", fc.GotAction)
	}
}

func TestCallRejectsNilResponse(t *testing.T) {
	t.Parallel()
	fc := &testutil.FakeCaller{Resp: nil, Err: nil}
	_, err := kaswrite.Call(context.Background(), fc, "session", "delete_session", nil)
	if err == nil {
		t.Fatal("Call err = nil, want error for nil response without error from Caller")
	}
	if errors.Is(err, kaswrite.ErrUnexpectedReturnString) {
		t.Errorf("nil-response err = %v, must NOT be ErrUnexpectedReturnString", err)
	}
	if !strings.Contains(err.Error(), "session") || !strings.Contains(err.Error(), "delete_session") {
		t.Errorf("err = %q, want it to contain label+action", err)
	}
}

func TestCallRejectsNonTrueReturnString(t *testing.T) {
	t.Parallel()
	resp := &soap.Response{Body: soap.ResponseBody{ReturnString: "FALSE"}}
	fc := &testutil.FakeCaller{Resp: resp}
	_, err := kaswrite.Call(context.Background(), fc, "mailinglist", "update_mailinglist", nil)
	if !errors.Is(err, kaswrite.ErrUnexpectedReturnString) {
		t.Fatalf("Call err = %v, want errors.Is ErrUnexpectedReturnString", err)
	}
	if !strings.Contains(err.Error(), "update_mailinglist") || !strings.Contains(err.Error(), "FALSE") {
		t.Errorf("err = %q, want it to contain the action and the observed value", err)
	}
}

func TestCallPropagatesCallerError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	fc := &testutil.FakeCaller{Err: want}
	_, err := kaswrite.Call(context.Background(), fc, "mailforward", "add_mailforward", nil)
	if !errors.Is(err, want) {
		t.Fatalf("Call err = %v, want %v wrapped", err, want)
	}
	if errors.Is(err, kaswrite.ErrUnexpectedReturnString) {
		t.Errorf("transport err = %v, must NOT be ErrUnexpectedReturnString", err)
	}
}
