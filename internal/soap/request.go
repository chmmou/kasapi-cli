package soap

import (
	"encoding/json"
	"fmt"
	"io"
)

// AuthType enumerates the kas_auth_type values the KAS API accepts.
type AuthType string

// Auth types per https://kasapi.kasserver.com/dokumentation/phpdoc/.
const (
	AuthPlain   AuthType = "plain"
	AuthSession AuthType = "session"
)

// Request is the typed payload for a KasApi call. It is encoded as JSON
// inside the <Params> element of the SOAP envelope.
type Request struct {
	Login    string
	AuthType AuthType
	AuthData string
	Action   string
	Params   map[string]any
}

const requestTemplate = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:tns="https://kasserver.com/">
    <soapenv:Body>
        <tns:KasApi>
            <Params>%s</Params>
        </tns:KasApi>
    </soapenv:Body>
</soapenv:Envelope>`

// EncodeRequest writes a SOAP request envelope for a KasApi call. The
// payload is serialized as JSON; json.Marshal HTML-escapes <, > and &, so
// the JSON body is always safe inside the XML <Params> element.
func EncodeRequest(w io.Writer, r Request) error {
	if r.Action == "" {
		return fmt.Errorf("soap: Request.Action is required")
	}
	if r.Login == "" {
		return fmt.Errorf("soap: Request.Login is required")
	}
	if r.AuthType == "" {
		return fmt.Errorf("soap: Request.AuthType is required")
	}
	params := r.Params
	if params == nil {
		params = map[string]any{}
	}
	payload := map[string]any{
		"KasRequestParams": params,
		"kas_action":       r.Action,
		"kas_auth_data":    r.AuthData,
		"kas_auth_type":    string(r.AuthType),
		"kas_login":        r.Login,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("soap: marshal params: %w", err)
	}
	_, err = fmt.Fprintf(w, requestTemplate, body)
	return err
}
