package soap

import "fmt"

// DecodeArray decodes a KAS ReturnInfo "Array of Maps" into a typed
// slice. Read modules build their typed list (e.g. DDNSUserList) by
// converting the returned []T at the call site.
//
// label is the module-name prefix used in error messages (e.g.
// "ddns", "softwareinstall"). mapper converts a single Map-valued
// Value into T; it is invoked once per item after this function has
// already verified that the item is a Map, so mappers do not need to
// re-check the kind.
//
// An error is returned when v is not an Array, or when any item is
// not a Map. The returned slice has capacity len(v.Array), so the
// typed conversion at the call site is allocation-free.
func DecodeArray[T any](v Value, label string, mapper func(Value) T) ([]T, error) {
	if v.Kind != KindArray {
		return nil, fmt.Errorf("%s: expected ReturnInfo array, got kind %d", label, v.Kind)
	}
	out := make([]T, 0, len(v.Array))
	for i, item := range v.Array {
		if item.Kind != KindMap {
			return nil, fmt.Errorf("%s: ReturnInfo[%d] is not a Map", label, i)
		}
		out = append(out, mapper(item))
	}
	return out, nil
}
