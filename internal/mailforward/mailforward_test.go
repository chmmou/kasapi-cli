package mailforward_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/mailforward"
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
	path := filepath.Join(repoRoot(t), "testdata", "mailforward", name)
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

func TestDecodeMailForwards(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_mailforwards_response_success.xml")
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
	resp := decodeFixture(t, "get_mailforward_response_success.xml")
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
	resp := decodeFixture(t, "get_mailforwards_response_success.xml")
	fc := &fakeCaller{resp: resp}
	list, err := mailforward.NewClient(fc).List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if fc.gotAction != "get_mailforwards" {
		t.Errorf("action = %q, want get_mailforwards", fc.gotAction)
	}
	if fc.gotParams != nil {
		t.Errorf("params = %v, want nil", fc.gotParams)
	}
	if len(list) != 2 {
		t.Errorf("len = %d, want 2", len(list))
	}
}

func TestClientGet(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_mailforward_response_success.xml")
	fc := &fakeCaller{resp: resp}
	f, err := mailforward.NewClient(fc).Get(context.Background(), "from@example.de")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if fc.gotAction != "get_mailforwards" {
		t.Errorf("action = %q, want get_mailforwards", fc.gotAction)
	}
	if got, _ := fc.gotParams["mail_forward"].(string); got != "from@example.de" {
		t.Errorf("params[mail_forward] = %v, want from@example.de", fc.gotParams["mail_forward"])
	}
	if f.Address == "" {
		t.Errorf("Address empty")
	}
}

func TestClientGetEmptyAddress(t *testing.T) {
	t.Parallel()
	c := mailforward.NewClient(&fakeCaller{})
	if _, err := c.Get(context.Background(), ""); err == nil {
		t.Errorf("Get(\"\") err = nil, want validation error")
	}
}

func TestClientGetNotFound(t *testing.T) {
	t.Parallel()
	resp := &soap.Response{Body: soap.ResponseBody{ReturnInfo: soap.Value{Kind: soap.KindArray}}}
	c := mailforward.NewClient(&fakeCaller{resp: resp})
	if _, err := c.Get(context.Background(), "missing@example.de"); err == nil {
		t.Errorf("Get on empty result err = nil, want not-found")
	}
}

func TestClientPropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := mailforward.NewClient(&fakeCaller{err: want})
	if _, err := c.List(context.Background()); !errors.Is(err, want) {
		t.Errorf("List err = %v, want %v wrapped", err, want)
	}
	if _, err := c.Get(context.Background(), "from@example.de"); !errors.Is(err, want) {
		t.Errorf("Get err = %v, want %v wrapped", err, want)
	}
}

func TestMailForwardListTabular(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_mailforwards_response_success.xml")
	list, _ := mailforward.DecodeMailForwards(resp.Body.ReturnInfo)
	rows := list.TableRows()
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	if rows[0][0] != "from@example.de" || rows[0][1] != "to@example.de" {
		t.Errorf("rows[0] = %v", rows[0])
	}
}
