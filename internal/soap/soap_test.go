package soap_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

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

func isFaultEnvelope(t *testing.T, path string) bool {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return strings.Contains(string(b), "SOAP-ENV:Fault")
}

func decodeFile(t *testing.T, path string) (*soap.Response, error) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer func() { _ = f.Close() }()
	return soap.Decode(f)
}

// TestDecodeAllResponseFixtures walks every response fixture under
// testdata/ (excluding testdata/session/, which is KasAuth) and dispatches
// by content: fixtures that contain SOAP-ENV:Fault must produce a
// *FaultError; everything else must decode into a populated Response.
func TestDecodeAllResponseFixtures(t *testing.T) {
	root := filepath.Join(repoRoot(t), "testdata")
	sessionPath := filepath.Join(root, "session") + string(filepath.Separator)
	var paths []string
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if strings.HasPrefix(p, sessionPath) {
			return nil
		}
		base := filepath.Base(p)
		if !strings.HasSuffix(base, ".xml") {
			return nil
		}
		if !strings.Contains(base, "_response_") && !strings.HasPrefix(base, "response_failed_") {
			return nil
		}
		paths = append(paths, p)
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("no response fixtures found")
	}
	for _, p := range paths {
		rel, _ := filepath.Rel(repoRoot(t), p)
		t.Run(rel, func(t *testing.T) {
			expectFault := isFaultEnvelope(t, p)
			resp, err := decodeFile(t, p)
			if expectFault {
				var fe *soap.FaultError
				if !errors.As(err, &fe) {
					t.Fatalf("expected *FaultError, got %T: %v", err, err)
				}
				if fe.Fault.String == "" {
					t.Errorf("faultstring is empty")
				}
				return
			}
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			if resp.Body.KasFloodDelay <= 0 {
				t.Errorf("KasFloodDelay = %v, want > 0", resp.Body.KasFloodDelay)
			}
			if resp.Body.ReturnString == "" {
				t.Errorf("ReturnString empty")
			}
			if resp.Request.Kind != soap.KindMap {
				t.Errorf("Request kind = %d, want Map", resp.Request.Kind)
			}
		})
	}
}

// TestDecodeGetAccountsShape pins the most-used response fixture: it must
// produce a 4-element array of account maps with the documented columns.
func TestDecodeGetAccountsShape(t *testing.T) {
	resp, err := decodeFile(t, filepath.Join(repoRoot(t), "testdata/account/get_accounts_response_success.xml"))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got, want := resp.Body.ReturnString, "TRUE"; got != want {
		t.Errorf("ReturnString = %q, want %q", got, want)
	}
	if resp.Body.ReturnInfo.Kind != soap.KindArray {
		t.Fatalf("ReturnInfo kind = %d, want Array", resp.Body.ReturnInfo.Kind)
	}
	if got := len(resp.Body.ReturnInfo.Array); got != 4 {
		t.Fatalf("len(ReturnInfo) = %d, want 4", got)
	}
	first := resp.Body.ReturnInfo.Array[0]
	login, ok := first.Get("account_login")
	if !ok || login.Kind != soap.KindString || login.String != "w0000001" {
		t.Errorf("first.account_login = %+v, want xsd:string w0000001", login)
	}
}

// TestDecodeGetServerInformationShape exercises the array-of-maps shape
// where ReturnInfo lists installed services.
func TestDecodeGetServerInformationShape(t *testing.T) {
	resp, err := decodeFile(t, filepath.Join(repoRoot(t), "testdata/account/get_server_information_response_success.xml"))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if resp.Body.ReturnInfo.Kind != soap.KindArray {
		t.Fatalf("ReturnInfo kind = %d, want Array", resp.Body.ReturnInfo.Kind)
	}
	if got := len(resp.Body.ReturnInfo.Array); got != 8 {
		t.Errorf("len(ReturnInfo) = %d, want 8", got)
	}
	mysql := resp.Body.ReturnInfo.Array[0]
	svc, _ := mysql.Get("service")
	if svc.AsString() != "mysql" {
		t.Errorf("first.service = %q, want mysql", svc.AsString())
	}
}

// TestDecodeFaultDetail verifies that fault fixtures expose faultstring and
// detail correctly.
func TestDecodeFaultDetail(t *testing.T) {
	_, err := decodeFile(t, filepath.Join(repoRoot(t), "testdata/response_failed_no_auth.xml"))
	var fe *soap.FaultError
	if !errors.As(err, &fe) {
		t.Fatalf("expected *FaultError, got %v", err)
	}
	if fe.Fault.String != "no_auth" {
		t.Errorf("faultstring = %q, want %q", fe.Fault.String, "no_auth")
	}
	if !strings.Contains(fe.Fault.Detail, "kas_login") {
		t.Errorf("detail = %q, want it to mention kas_login", fe.Fault.Detail)
	}
}

// TestDecodeEmptyArray covers the self-closing
// <value SOAP-ENC:arrayType="xsd:ur-type[0]" xsi:type="SOAP-ENC:Array"/>
// case (empty KasRequestParams in the echoed request).
func TestDecodeEmptyArray(t *testing.T) {
	resp, err := decodeFile(t, filepath.Join(repoRoot(t), "testdata/account/get_accounts_response_success.xml"))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	params, ok := resp.Request.Get("KasRequestParams")
	if !ok {
		t.Fatal("KasRequestParams not found")
	}
	if params.Kind != soap.KindArray {
		t.Errorf("KasRequestParams kind = %d, want Array", params.Kind)
	}
	if len(params.Array) != 0 {
		t.Errorf("expected empty array, got %d elements", len(params.Array))
	}
}

// TestEncodeRequestRoundtrip verifies the encoder produces a parseable
// envelope and that the JSON payload contains the expected fields.
func TestEncodeRequestRoundtrip(t *testing.T) {
	var buf bytes.Buffer
	err := soap.EncodeRequest(&buf, soap.Request{
		Login:    "w0000000",
		AuthType: soap.AuthSession,
		AuthData: "REDACTED",
		Action:   "get_accounts",
		Params:   nil,
	})
	if err != nil {
		t.Fatalf("EncodeRequest: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		`<?xml version="1.0"`,
		`<tns:KasApi>`,
		`"kas_login":"w0000000"`,
		`"kas_action":"get_accounts"`,
		`"kas_auth_type":"session"`,
		`"kas_auth_data":"REDACTED"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("encoded request missing %q\n--- output ---\n%s", want, out)
		}
	}
}

// TestEncodeRequestRequiresFields verifies that EncodeRequest rejects
// malformed input rather than silently producing an unauthenticated call.
func TestEncodeRequestRequiresFields(t *testing.T) {
	cases := []struct {
		name string
		req  soap.Request
	}{
		{"no action", soap.Request{Login: "w0000000", AuthType: soap.AuthSession}},
		{"no login", soap.Request{Action: "x", AuthType: soap.AuthSession}},
		{"no auth type", soap.Request{Action: "x", Login: "w0000000"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := soap.EncodeRequest(&bytes.Buffer{}, tc.req)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
