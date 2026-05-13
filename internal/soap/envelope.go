package soap

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
)

// MaxResponseBytes caps the byte count the KAS-API decoders will read
// from a single response body. Real KAS responses are well under 1 MB
// (the largest captured fixture is ~70 KB); 16 MB is comfortable
// headroom while still preventing a pathological response (compromised
// endpoint, MITM, server bug) from exhausting memory through a
// streaming XML payload that never ends.
const MaxResponseBytes = 16 << 20

// Response is the parsed body of a successful KasApi SOAP envelope. It
// keeps both the typed shortcuts (KasFloodDelay, ReturnString, ReturnInfo)
// and the full ordered Map for callers needing extras.
type Response struct {
	Request   Value
	Body      ResponseBody
	RawReturn Value
}

// ResponseBody mirrors the canonical fields under <return><Response>.
type ResponseBody struct {
	KasFloodDelay float64
	ReturnString  string
	ReturnInfo    Value
	Msg           Value
	Raw           []KV
}

// Fault is the parsed payload of a SOAP-ENV:Fault element.
type Fault struct {
	Code   string
	String string
	Actor  string
	Detail string
}

// FaultError wraps a Fault so it can be returned as a Go error.
type FaultError struct {
	Fault Fault
}

func (e *FaultError) Error() string {
	if e.Fault.Detail != "" {
		return fmt.Sprintf("kas-api fault %q: %s", e.Fault.String, e.Fault.Detail)
	}
	return fmt.Sprintf("kas-api fault %q", e.Fault.String)
}

// Decode parses a KasApi SOAP envelope. It returns a typed Response on
// success and a *FaultError when the body contained a SOAP-ENV:Fault.
//
// The reader is wrapped in an io.LimitReader (MaxResponseBytes) and the
// decoder runs in Strict mode (the encoding/xml default, set explicitly
// here so a future refactor cannot silently flip it). The KAS server is
// trusted, so this is defense-in-depth against a malformed or hostile
// response rather than a security boundary.
func Decode(r io.Reader) (*Response, error) {
	dec := xml.NewDecoder(io.LimitReader(r, MaxResponseBytes))
	dec.Strict = true
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return nil, errors.New("soap: empty document")
		}
		if err != nil {
			return nil, err
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if start.Name.Local == "Body" {
			return decodeBody(dec, start)
		}
	}
}

func decodeBody(d *xml.Decoder, parent xml.StartElement) (*Response, error) {
	for {
		tok, err := d.Token()
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "KasApiResponse":
				return decodeKasApiResponse(d, t)
			case "Fault":
				fault, err := DecodeFault(d, t)
				if err != nil {
					return nil, err
				}
				return nil, &FaultError{Fault: *fault}
			default:
				if err := d.Skip(); err != nil {
					return nil, err
				}
			}
		case xml.EndElement:
			if t.Name == parent.Name {
				return nil, errors.New("soap: empty Body")
			}
		}
	}
}

func decodeKasApiResponse(d *xml.Decoder, parent xml.StartElement) (*Response, error) {
	for {
		tok, err := d.Token()
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "return" {
				var v Value
				if err := v.UnmarshalXML(d, t); err != nil {
					return nil, err
				}
				return buildResponse(v)
			}
			if err := d.Skip(); err != nil {
				return nil, err
			}
		case xml.EndElement:
			if t.Name == parent.Name {
				return nil, errors.New("soap: missing <return> element")
			}
		}
	}
}

func buildResponse(top Value) (*Response, error) {
	if top.Kind != KindMap {
		return nil, fmt.Errorf("soap: <return> is not a Map (kind=%d)", top.Kind)
	}
	out := &Response{RawReturn: top}
	for _, kv := range top.Map {
		switch kv.Key {
		case "Request":
			out.Request = kv.Value
		case "Response":
			body, err := buildResponseBody(kv.Value)
			if err != nil {
				return nil, err
			}
			out.Body = body
		}
	}
	return out, nil
}

func buildResponseBody(v Value) (ResponseBody, error) {
	var out ResponseBody
	if v.Kind != KindMap {
		return out, fmt.Errorf("soap: Response is not a Map (kind=%d)", v.Kind)
	}
	out.Raw = v.Map
	for _, kv := range v.Map {
		switch kv.Key {
		case "KasFloodDelay":
			out.KasFloodDelay = kv.Value.AsFloat()
		case "ReturnString":
			out.ReturnString = kv.Value.AsString()
		case "ReturnInfo":
			out.ReturnInfo = kv.Value
		case "Msg":
			out.Msg = kv.Value
		}
	}
	return out, nil
}

// DecodeFault parses a <SOAP-ENV:Fault> element. The decoder must be
// positioned just past the start element, which is passed as parent so
// the function knows when to stop. It is exported so other packages
// (e.g. auth) can decode KAS faults from their own envelope layouts.
func DecodeFault(d *xml.Decoder, parent xml.StartElement) (*Fault, error) {
	out := &Fault{}
	for {
		tok, err := d.Token()
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			s, err := readCharData(d, t)
			if err != nil {
				return nil, err
			}
			switch t.Name.Local {
			case "faultcode":
				out.Code = s
			case "faultstring":
				out.String = s
			case "faultactor":
				out.Actor = s
			case "detail":
				out.Detail = s
			}
		case xml.EndElement:
			if t.Name == parent.Name {
				return out, nil
			}
		}
	}
}
