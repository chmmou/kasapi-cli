package mailforward_test

import (
	"context"
	"errors"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/mailforward"
	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestDecodeMailForwards(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "mailforward/get_mailforwards_response_success.xml")
	got, err := mailforward.DecodeMailForwards(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeMailForwards: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (per fixture arrayType)", len(got))
	}
	f := got[0]
	if f.Address != "from@example.de" || f.Adress != "from@example.de" {
		t.Errorf("address pair = %q / %q", f.Address, f.Adress)
	}
	if f.Targets != "to@example.de" {
		t.Errorf("Targets = %q", f.Targets)
	}
	if f.Spamfilter != "kaspdw" {
		t.Errorf("Spamfilter = %q", f.Spamfilter)
	}
	if f.InProgress != "FALSE" {
		t.Errorf("InProgress = %q", f.InProgress)
	}
}

func TestDecodeMailForwardSingular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "mailforward/get_mailforward_response_success.xml")
	got, err := mailforward.DecodeMailForwards(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeMailForwards: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Address == "" {
		t.Errorf("Address empty")
	}
}

func TestClientList(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "mailforward/get_mailforwards_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	list, err := mailforward.NewClient(fc).List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if fc.GotAction != "get_mailforwards" {
		t.Errorf("action = %q, want get_mailforwards", fc.GotAction)
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
	resp := testutil.DecodeFixture(t, "mailforward/get_mailforward_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	f, err := mailforward.NewClient(fc).Get(context.Background(), "from@example.de")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if fc.GotAction != "get_mailforwards" {
		t.Errorf("action = %q, want get_mailforwards", fc.GotAction)
	}
	if got, _ := fc.GotParams["mail_forward"].(string); got != "from@example.de" {
		t.Errorf("params[mail_forward] = %v, want from@example.de", fc.GotParams["mail_forward"])
	}
	if f.Address == "" {
		t.Errorf("Address empty")
	}
}

func TestClientGetEmptyAddress(t *testing.T) {
	t.Parallel()
	c := mailforward.NewClient(&testutil.FakeCaller{})
	if _, err := c.Get(context.Background(), ""); err == nil {
		t.Errorf("Get(\"\") err = nil, want validation error")
	}
}

func TestClientGetNotFound(t *testing.T) {
	t.Parallel()
	resp := &soap.Response{Body: soap.ResponseBody{ReturnInfo: soap.Value{Kind: soap.KindArray}}}
	c := mailforward.NewClient(&testutil.FakeCaller{Resp: resp})
	if _, err := c.Get(context.Background(), "missing@example.de"); err == nil {
		t.Errorf("Get on empty result err = nil, want not-found")
	}
}

func TestClientPropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := mailforward.NewClient(&testutil.FakeCaller{Err: want})
	if _, err := c.List(context.Background()); !errors.Is(err, want) {
		t.Errorf("List err = %v, want %v wrapped", err, want)
	}
	if _, err := c.Get(context.Background(), "from@example.de"); !errors.Is(err, want) {
		t.Errorf("Get err = %v, want %v wrapped", err, want)
	}
}

func TestMailForwardListTabular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "mailforward/get_mailforwards_response_success.xml")
	list, _ := mailforward.DecodeMailForwards(resp.Body.ReturnInfo)
	rows := list.TableRows()
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	if rows[0][0] != "from@example.de" || rows[0][1] != "to@example.de" {
		t.Errorf("rows[0] = %v", rows[0])
	}
}
