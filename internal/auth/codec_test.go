package auth_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/auth"
	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func loadFixture(t *testing.T, rel string) []byte {
	t.Helper()
	//nolint:gosec // G304: test fixture loader, path is rooted at testutil.RepoRoot(t).
	b, err := os.ReadFile(filepath.Join(testutil.RepoRoot(t), "testdata", rel))
	if err != nil {
		t.Fatalf("load %s: %v", rel, err)
	}
	return b
}

func TestEncodeRequestRequiredFields(t *testing.T) {
	cases := []auth.Request{
		{AuthType: soap.AuthPlain, AuthData: "x"}, // missing Login
		{Login: "w0", AuthData: "x"},              // missing AuthType
		{Login: "w0", AuthType: soap.AuthPlain},   // missing AuthData
	}
	for i, r := range cases {
		var buf bytes.Buffer
		if err := auth.EncodeRequest(&buf, r); err == nil {
			t.Errorf("case %d: expected error", i)
		}
	}
}

func TestEncodeRequestOptionalFields(t *testing.T) {
	yes := true
	r := auth.Request{
		Login:          "w0000000",
		AuthType:       soap.AuthPlain,
		AuthData:       "secret",
		Lifetime:       1800,
		UpdateLifetime: &yes,
		OTP:            "123456",
	}
	var buf bytes.Buffer
	if err := auth.EncodeRequest(&buf, r); err != nil {
		t.Fatalf("EncodeRequest: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "<tns:KasAuth>") {
		t.Error("envelope missing <tns:KasAuth>")
	}
	// Extract JSON between <Params> and </Params> and round-trip it so
	// we can assert on field presence without depending on key order.
	start := strings.Index(out, "<Params>") + len("<Params>")
	end := strings.Index(out, "</Params>")
	var payload map[string]any
	if err := json.Unmarshal([]byte(out[start:end]), &payload); err != nil {
		t.Fatalf("json: %v\npayload=%q", err, out[start:end])
	}
	for _, k := range []string{"kas_login", "kas_auth_type", "kas_auth_data",
		"session_lifetime", "session_update_lifetime", "session_2fa"} {
		if _, ok := payload[k]; !ok {
			t.Errorf("payload missing %q: %v", k, payload)
		}
	}
	if payload["session_update_lifetime"] != "Y" {
		t.Errorf("session_update_lifetime = %v, want Y", payload["session_update_lifetime"])
	}
}

func TestEncodeRequestOmitsZeroOptionals(t *testing.T) {
	r := auth.Request{Login: "w0", AuthType: soap.AuthPlain, AuthData: "secret"}
	var buf bytes.Buffer
	if err := auth.EncodeRequest(&buf, r); err != nil {
		t.Fatalf("EncodeRequest: %v", err)
	}
	out := buf.String()
	for _, k := range []string{"session_lifetime", "session_update_lifetime", "session_2fa"} {
		if strings.Contains(out, k) {
			t.Errorf("payload should omit %q: %s", k, out)
		}
	}
}

func TestDecodeResponseSuccessFixture(t *testing.T) {
	body := loadFixture(t, "session/add_session_response_success.xml")
	tok, err := auth.DecodeResponse(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("DecodeResponse: %v", err)
	}
	if tok != "01234567890abcdef0123456789abcdef0123456" {
		t.Errorf("token = %q", tok)
	}
}

func TestDecodeResponseFaultFixture(t *testing.T) {
	body := loadFixture(t, "session/add_session_response_failed_otp_pin_incorrect.xml")
	_, err := auth.DecodeResponse(bytes.NewReader(body))
	if err == nil {
		t.Fatal("expected error")
	}
	var fe *soap.FaultError
	if !errors.As(err, &fe) {
		t.Fatalf("expected *soap.FaultError, got %T", err)
	}
	if fe.Fault.String != "otp_pin_incorrect" {
		t.Errorf("fault = %q, want otp_pin_incorrect", fe.Fault.String)
	}
}

func TestDecodeResponseEmptyDocument(t *testing.T) {
	if _, err := auth.DecodeResponse(strings.NewReader("")); err == nil {
		t.Fatal("expected error")
	}
}

// The credential token contract is 40 alphanumeric characters; anything
// else must be rejected before it gets cached and persisted. The error
// must not echo the token content — only its length.
func TestDecodeResponseRejectsMalformedToken(t *testing.T) {
	const envelope = `<?xml version="1.0" encoding="UTF-8"?>
<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/">
    <SOAP-ENV:Body>
        <ns1:KasAuthResponse xmlns:ns1="https://kasserver.com/">
            <return xsi:type="xsd:string" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">%s</return>
        </ns1:KasAuthResponse>
    </SOAP-ENV:Body>
</SOAP-ENV:Envelope>`
	for _, tok := range []string{
		"short",
		strings.Repeat("a", 41),
		strings.Repeat("a", 39) + "!",
	} {
		_, err := auth.DecodeResponse(strings.NewReader(fmt.Sprintf(envelope, tok)))
		if err == nil {
			t.Errorf("token %q: expected malformed-token error, got nil", tok)
			continue
		}
		if strings.Contains(err.Error(), tok) {
			t.Errorf("token content leaked into error: %v", err)
		}
	}
}

// session_lifetime is documented as 1..30000 seconds; 0 means "leave
// the server default". Out-of-range values must fail at encode time so
// the local expiry mirror cannot silently diverge from the server.
func TestEncodeRequestRejectsOutOfRangeLifetime(t *testing.T) {
	for _, lifetime := range []int{-1, 30001} {
		var buf bytes.Buffer
		err := auth.EncodeRequest(&buf, auth.Request{
			Login:    "w0",
			AuthType: soap.AuthPlain,
			AuthData: "pw",
			Lifetime: lifetime,
		})
		if err == nil {
			t.Errorf("Lifetime %d: expected range error, got nil", lifetime)
		}
	}
}
