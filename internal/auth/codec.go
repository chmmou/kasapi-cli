package auth

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"

	"github.com/chmmou/kasapi-cli/internal/soap"
)

// Request is the typed payload for a KasAuth call. Lifetime and
// UpdateLifetime / OTP map to session_lifetime, session_update_lifetime,
// and session_2fa per the KasAuth documentation. Zero-valued optionals
// are omitted from the encoded JSON so the server applies its defaults.
type Request struct {
	Login          string
	AuthType       soap.AuthType
	AuthData       string
	Lifetime       int
	UpdateLifetime *bool
	OTP            string
}

// maxSessionLifetime is the documented upper bound of the KasAuth
// session_lifetime parameter (seconds); the documented range is
// 1..30000, with an unset value (0 here) leaving the server default.
const maxSessionLifetime = 30000

// credentialTokenLength is the length of the credential token a
// successful KasAuth call returns (see doc.go): 40 alphanumeric
// characters.
const credentialTokenLength = 40

const requestTemplate = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:tns="https://kasserver.com/">
    <soapenv:Body>
        <tns:KasAuth>
            <Params>%s</Params>
        </tns:KasAuth>
    </soapenv:Body>
</soapenv:Envelope>`

// EncodeRequest writes a SOAP request envelope for a KasAuth call.
func EncodeRequest(w io.Writer, r Request) error {
	if r.Login == "" {
		return errors.New("auth: Request.Login is required")
	}
	if r.AuthType == "" {
		return errors.New("auth: Request.AuthType is required")
	}
	if r.AuthData == "" {
		return errors.New("auth: Request.AuthData is required")
	}
	if r.Lifetime < 0 || r.Lifetime > maxSessionLifetime {
		return fmt.Errorf("auth: Request.Lifetime %d out of range 1..%d (0 = server default)", r.Lifetime, maxSessionLifetime)
	}
	payload := map[string]any{
		"kas_login":     r.Login,
		"kas_auth_type": string(r.AuthType),
		"kas_auth_data": r.AuthData,
	}
	if r.Lifetime > 0 {
		payload["session_lifetime"] = r.Lifetime
	}
	if r.UpdateLifetime != nil {
		if *r.UpdateLifetime {
			payload["session_update_lifetime"] = "Y"
		} else {
			payload["session_update_lifetime"] = "N"
		}
	}
	if r.OTP != "" {
		payload["session_2fa"] = r.OTP
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("auth: marshal params: %w", err)
	}
	_, err = fmt.Fprintf(w, requestTemplate, body)
	return err
}

// DecodeResponse parses a KasAuth SOAP envelope. On success it returns
// the credential token; on a SOAP-ENV:Fault body it returns
// *soap.FaultError so callers can wrap it into auth.Error.
//
// The reader is wrapped in an io.LimitReader (soap.MaxResponseBytes)
// and Strict mode is set explicitly; see soap.Decode for the rationale.
func DecodeResponse(r io.Reader) (string, error) {
	dec := xml.NewDecoder(io.LimitReader(r, soap.MaxResponseBytes))
	dec.Strict = true
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return "", errors.New("auth: empty document")
		}
		if err != nil {
			return "", err
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

func decodeBody(d *xml.Decoder, parent xml.StartElement) (string, error) {
	for {
		tok, err := d.Token()
		if err != nil {
			return "", err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "KasAuthResponse":
				return decodeKasAuthResponse(d, t)
			case "Fault":
				fault, err := soap.DecodeFault(d, t)
				if err != nil {
					return "", err
				}
				return "", &soap.FaultError{Fault: *fault}
			default:
				if err := d.Skip(); err != nil {
					return "", err
				}
			}
		case xml.EndElement:
			if t.Name == parent.Name {
				return "", errors.New("auth: empty Body")
			}
		}
	}
}

// validToken reports whether s has the shape of a KasAuth credential
// token: exactly credentialTokenLength alphanumeric characters.
func validToken(s string) bool {
	if len(s) != credentialTokenLength {
		return false
	}
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
		default:
			return false
		}
	}
	return true
}

func decodeKasAuthResponse(d *xml.Decoder, parent xml.StartElement) (string, error) {
	for {
		tok, err := d.Token()
		if err != nil {
			return "", err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "return" {
				var s string
				if err := d.DecodeElement(&s, &t); err != nil {
					return "", err
				}
				if s == "" {
					return "", errors.New("auth: empty <return> element")
				}
				// Guard the token shape before it gets cached and
				// persisted; only the length is reported so a partial
				// secret can never leak into an error message.
				if !validToken(s) {
					return "", fmt.Errorf("auth: malformed credential token: got %d bytes, want %d alphanumeric characters", len(s), credentialTokenLength)
				}
				return s, nil
			}
			if err := d.Skip(); err != nil {
				return "", err
			}
		case xml.EndElement:
			if t.Name == parent.Name {
				return "", errors.New("auth: missing <return> element")
			}
		}
	}
}
