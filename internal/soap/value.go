package soap

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

// Kind identifies the runtime shape of a Value.
type Kind uint8

// Kinds enumerate the discriminator values KAS uses on xsi:type.
const (
	KindNil Kind = iota
	KindString
	KindInt
	KindFloat
	KindBool
	KindMap
	KindArray
)

// KV is one entry of an ordered Map. The Apache xml-soap ns2:Map preserves
// insertion order; we keep that order so callers can render or compare
// without surprise.
type KV struct {
	Key   string
	Value Value
}

// Value is the discriminated union for ns2:Map / SOAP-ENC:Array / xsd:*
// scalars in a KAS response.
type Value struct {
	Kind   Kind
	String string
	Int    int64
	Float  float64
	Bool   bool
	Map    []KV
	Array  []Value
}

// UnmarshalXML reads a single typed <value> (or <item>) element. The
// xsi:type attribute on start drives the dispatch.
func (v *Value) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	xsiType, isNil := readTypeAttrs(start.Attr)
	if isNil {
		v.Kind = KindNil
		return d.Skip()
	}
	switch classifyType(xsiType) {
	case KindString:
		s, err := readCharData(d, start)
		if err != nil {
			return err
		}
		v.Kind = KindString
		v.String = s
		return nil
	case KindInt:
		s, err := readCharData(d, start)
		if err != nil {
			return err
		}
		s = strings.TrimSpace(s)
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return fmt.Errorf("soap: invalid xsd:int %q: %w", s, err)
		}
		v.Kind = KindInt
		v.Int = n
		return nil
	case KindFloat:
		s, err := readCharData(d, start)
		if err != nil {
			return err
		}
		s = strings.TrimSpace(s)
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("soap: invalid xsd:float %q: %w", s, err)
		}
		v.Kind = KindFloat
		v.Float = f
		return nil
	case KindBool:
		s, err := readCharData(d, start)
		if err != nil {
			return err
		}
		s = strings.TrimSpace(strings.ToLower(s))
		switch s {
		case "true", "1":
			v.Bool = true
		case "false", "0", "":
			v.Bool = false
		default:
			return fmt.Errorf("soap: invalid xsd:boolean %q", s)
		}
		v.Kind = KindBool
		return nil
	case KindMap:
		v.Kind = KindMap
		return decodeMapItems(d, start, &v.Map)
	case KindArray:
		v.Kind = KindArray
		return decodeArrayItems(d, start, &v.Array)
	default:
		return fmt.Errorf("soap: unknown xsi:type %q on <%s>", xsiType, start.Name.Local)
	}
}

func decodeMapItems(d *xml.Decoder, parent xml.StartElement, out *[]KV) error {
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local != "item" {
				if err := d.Skip(); err != nil {
					return err
				}
				continue
			}
			kv, err := decodeMapItem(d, t)
			if err != nil {
				return err
			}
			*out = append(*out, kv)
		case xml.EndElement:
			if t.Name == parent.Name {
				return nil
			}
		}
	}
}

func decodeMapItem(d *xml.Decoder, start xml.StartElement) (KV, error) {
	var kv KV
	for {
		tok, err := d.Token()
		if err != nil {
			return kv, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "key":
				s, err := readCharData(d, t)
				if err != nil {
					return kv, err
				}
				kv.Key = strings.TrimSpace(s)
			case "value":
				var inner Value
				if err := inner.UnmarshalXML(d, t); err != nil {
					return kv, err
				}
				kv.Value = inner
			default:
				if err := d.Skip(); err != nil {
					return kv, err
				}
			}
		case xml.EndElement:
			if t.Name == start.Name {
				return kv, nil
			}
		}
	}
}

func decodeArrayItems(d *xml.Decoder, parent xml.StartElement, out *[]Value) error {
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local != "item" {
				if err := d.Skip(); err != nil {
					return err
				}
				continue
			}
			var v Value
			if err := v.UnmarshalXML(d, t); err != nil {
				return err
			}
			*out = append(*out, v)
		case xml.EndElement:
			if t.Name == parent.Name {
				return nil
			}
		}
	}
}

func readCharData(d *xml.Decoder, start xml.StartElement) (string, error) {
	var sb strings.Builder
	for {
		tok, err := d.Token()
		if err != nil {
			return "", err
		}
		switch t := tok.(type) {
		case xml.CharData:
			sb.Write(t)
		case xml.EndElement:
			if t.Name == start.Name {
				return sb.String(), nil
			}
		case xml.StartElement:
			if err := d.Skip(); err != nil {
				return "", err
			}
		}
	}
}

func readTypeAttrs(attrs []xml.Attr) (xsiType string, isNil bool) {
	for _, a := range attrs {
		switch a.Name.Local {
		case "type":
			xsiType = a.Value
		case "nil":
			if a.Value == "true" || a.Value == "1" {
				isNil = true
			}
		}
	}
	return xsiType, isNil
}

func classifyType(t string) Kind {
	if t == "" {
		return KindNil
	}
	_, local, found := strings.Cut(t, ":")
	if !found {
		local = t
	}
	switch local {
	case "string":
		return KindString
	case "int", "integer", "long", "short":
		return KindInt
	case "float", "double", "decimal":
		return KindFloat
	case "boolean":
		return KindBool
	case "Map":
		return KindMap
	case "Array":
		return KindArray
	}
	return Kind(255)
}

// Get looks up a key in a Map Value. It returns the zero Value and false if
// v is not a Map or the key is absent.
func (v Value) Get(key string) (Value, bool) {
	if v.Kind != KindMap {
		return Value{}, false
	}
	for _, kv := range v.Map {
		if kv.Key == key {
			return kv.Value, true
		}
	}
	return Value{}, false
}

// AsString coerces scalar kinds to their textual form. Maps and Arrays
// return the empty string.
func (v Value) AsString() string {
	switch v.Kind {
	case KindString:
		return v.String
	case KindInt:
		return strconv.FormatInt(v.Int, 10)
	case KindFloat:
		return strconv.FormatFloat(v.Float, 'g', -1, 64)
	case KindBool:
		return strconv.FormatBool(v.Bool)
	}
	return ""
}

// AsFloat coerces numeric kinds to float64. Strings are parsed when they
// represent a valid number; everything else returns 0.
func (v Value) AsFloat() float64 {
	switch v.Kind {
	case KindFloat:
		return v.Float
	case KindInt:
		return float64(v.Int)
	case KindString:
		f, err := strconv.ParseFloat(strings.TrimSpace(v.String), 64)
		if err != nil {
			return 0
		}
		return f
	}
	return 0
}

// AsInt coerces numeric kinds to int64. Strings are parsed when they
// represent a valid integer; floats truncate toward zero; everything
// else returns 0.
func (v Value) AsInt() int64 {
	switch v.Kind {
	case KindInt:
		return v.Int
	case KindFloat:
		return int64(v.Float)
	case KindString:
		s := strings.TrimSpace(v.String)
		if s == "" {
			return 0
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0
		}
		return n
	}
	return 0
}

// MapString looks up key in a Map Value and returns the entry coerced
// to its textual form via AsString. Missing keys, non-Map receivers,
// and xsi:nil values yield "". This is the canonical accessor read
// modules use to extract scalar string fields from a KAS response Map.
func (v Value) MapString(key string) string {
	inner, ok := v.Get(key)
	if !ok {
		return ""
	}
	return inner.AsString()
}

// MapInt looks up key in a Map Value and returns the entry coerced to
// int via AsInt. Missing keys and unparseable values yield 0.
func (v Value) MapInt(key string) int {
	return int(v.MapInt64(key))
}

// MapInt64 is the int64 form of MapInt for fields whose magnitude
// would not fit a 32-bit int on all targets (counters, byte sizes).
func (v Value) MapInt64(key string) int64 {
	inner, ok := v.Get(key)
	if !ok {
		return 0
	}
	return inner.AsInt()
}

// MapFloat looks up key in a Map Value and returns the entry coerced
// to float64 via AsFloat. Missing keys yield 0.
func (v Value) MapFloat(key string) float64 {
	inner, ok := v.Get(key)
	if !ok {
		return 0
	}
	return inner.AsFloat()
}
