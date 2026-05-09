package soap_test

import (
	"strings"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/soap"
)

type testItem struct {
	Name string
	Age  int
}

func mapItem(v soap.Value) testItem {
	return testItem{
		Name: v.MapString("name"),
		Age:  v.MapInt("age"),
	}
}

func mapValue(name string, age int) soap.Value {
	return soap.Value{
		Kind: soap.KindMap,
		Map: []soap.KV{
			{Key: "name", Value: soap.Value{Kind: soap.KindString, String: name}},
			{Key: "age", Value: soap.Value{Kind: soap.KindInt, Int: int64(age)}},
		},
	}
}

func TestDecodeArrayHappyPath(t *testing.T) {
	v := soap.Value{
		Kind: soap.KindArray,
		Array: []soap.Value{
			mapValue("alice", 30),
			mapValue("bob", 25),
		},
	}

	out, err := soap.DecodeArray(v, "test", mapItem)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len = %d, want 2", len(out))
	}
	if out[0] != (testItem{Name: "alice", Age: 30}) {
		t.Errorf("out[0] = %+v, want {alice 30}", out[0])
	}
	if out[1] != (testItem{Name: "bob", Age: 25}) {
		t.Errorf("out[1] = %+v, want {bob 25}", out[1])
	}
}

func TestDecodeArrayEmpty(t *testing.T) {
	v := soap.Value{Kind: soap.KindArray}
	out, err := soap.DecodeArray(v, "test", mapItem)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("len = %d, want 0", len(out))
	}
	if out == nil {
		t.Error("out should be a non-nil empty slice (KAS-empty != Go-nil)")
	}
}

func TestDecodeArrayWrongKind(t *testing.T) {
	v := soap.Value{Kind: soap.KindMap}
	_, err := soap.DecodeArray(v, "test", mapItem)
	if err == nil {
		t.Fatal("expected error for non-Array kind, got nil")
	}
	if !strings.Contains(err.Error(), "test:") {
		t.Errorf("error %q missing label prefix", err)
	}
	if !strings.Contains(err.Error(), "expected ReturnInfo array") {
		t.Errorf("error %q missing kind hint", err)
	}
}

func TestDecodeArrayItemNotMap(t *testing.T) {
	v := soap.Value{
		Kind: soap.KindArray,
		Array: []soap.Value{
			mapValue("alice", 30),
			{Kind: soap.KindString, String: "not-a-map"},
		},
	}
	_, err := soap.DecodeArray(v, "test", mapItem)
	if err == nil {
		t.Fatal("expected error for non-Map item, got nil")
	}
	if !strings.Contains(err.Error(), "ReturnInfo[1]") {
		t.Errorf("error %q missing item index", err)
	}
}
