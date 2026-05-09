package kasread_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/kasread"
	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/testutil"
)

type widget struct {
	Name string
}

type widgetList []widget

func decodeWidgets(v soap.Value) (widgetList, error) {
	if v.Kind != soap.KindArray {
		return nil, errors.New("not an array")
	}
	out := make(widgetList, 0, len(v.Array))
	for _, item := range v.Array {
		out = append(out, widget{Name: item.MapString("name")})
	}
	return out, nil
}

func arrayResp(names ...string) *soap.Response {
	arr := make([]soap.Value, 0, len(names))
	for _, n := range names {
		arr = append(arr, soap.Value{
			Kind: soap.KindMap,
			Map:  []soap.KV{{Key: "name", Value: soap.Value{Kind: soap.KindString, String: n}}},
		})
	}
	return &soap.Response{
		Body: soap.ResponseBody{
			ReturnInfo: soap.Value{Kind: soap.KindArray, Array: arr},
		},
	}
}

func newLG(fc *testutil.FakeCaller) kasread.ListGet[widgetList, widget] {
	return kasread.ListGet[widgetList, widget]{
		Caller:    fc,
		Action:    "get_widgets",
		Label:     "widget",
		ArgName:   "name",
		FilterKey: "widget_name",
		Decoder:   decodeWidgets,
	}
}

func TestListHappyPath(t *testing.T) {
	fc := &testutil.FakeCaller{Resp: arrayResp("alice", "bob")}
	got, err := newLG(fc).List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if fc.GotAction != "get_widgets" {
		t.Errorf("action = %q, want get_widgets", fc.GotAction)
	}
	if fc.GotParams != nil {
		t.Errorf("params = %v, want nil", fc.GotParams)
	}
	if len(got) != 2 || got[0].Name != "alice" {
		t.Errorf("got = %+v", got)
	}
}

func TestListPropagatesTransportError(t *testing.T) {
	want := errors.New("boom")
	fc := &testutil.FakeCaller{Err: want}
	if _, err := newLG(fc).List(context.Background()); !errors.Is(err, want) {
		t.Errorf("err = %v, want errors.Is == %v", err, want)
	}
}

func TestListWrapsDecodeError(t *testing.T) {
	fc := &testutil.FakeCaller{Resp: &soap.Response{Body: soap.ResponseBody{ReturnInfo: soap.Value{Kind: soap.KindMap}}}}
	_, err := newLG(fc).List(context.Background())
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
	if !strings.HasPrefix(err.Error(), "widget: get_widgets:") {
		t.Errorf("err = %q, want prefix 'widget: get_widgets:'", err)
	}
}

func TestGetHappyPath(t *testing.T) {
	fc := &testutil.FakeCaller{Resp: arrayResp("alice")}
	got, err := newLG(fc).Get(context.Background(), "alice")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if v, ok := fc.GotParams["widget_name"]; !ok || v != "alice" {
		t.Errorf("widget_name param = %v (ok=%v), want alice", v, ok)
	}
	if got.Name != "alice" {
		t.Errorf("got.Name = %q, want alice", got.Name)
	}
}

func TestGetEmptyValue(t *testing.T) {
	fc := &testutil.FakeCaller{}
	_, err := newLG(fc).Get(context.Background(), "")
	if err == nil {
		t.Fatal("expected required-arg error, got nil")
	}
	if err.Error() != "widget: name is required" {
		t.Errorf("err = %q, want exact 'widget: name is required'", err)
	}
	if fc.GotAction != "" {
		t.Errorf("FakeCaller was called for empty input: action=%q", fc.GotAction)
	}
}

func TestGetNotFound(t *testing.T) {
	fc := &testutil.FakeCaller{Resp: arrayResp()}
	_, err := newLG(fc).Get(context.Background(), "ghost")
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
	if !strings.Contains(err.Error(), `"ghost" not found`) {
		t.Errorf("err = %q, want it to contain '%q not found'", err, "ghost")
	}
}

func TestGetPropagatesTransportError(t *testing.T) {
	want := errors.New("boom")
	fc := &testutil.FakeCaller{Err: want}
	if _, err := newLG(fc).Get(context.Background(), "alice"); !errors.Is(err, want) {
		t.Errorf("err = %v, want errors.Is == %v", err, want)
	}
}
