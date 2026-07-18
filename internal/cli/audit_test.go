package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chmmou/kasapi-cli/internal/api"
	"github.com/chmmou/kasapi-cli/internal/cli"
)

func TestRedactParams(t *testing.T) {
	t.Parallel()
	in := map[string]any{
		"mail_login":    "m0000001",
		"mail_password": "hunter2",
		"auth_data":     "topsecret",
		"new_password":  "p4ss",
		"api_token":     "abc123",   // substring rule
		"db_secret":     "shh",      // substring rule
		"ftp_passwort":  "geheim",   // German spelling, substring rule
		"db_passwort":   "geheim2",  // German spelling, substring rule
		"record_id":     42,         // non-string, kept
		"comment":       "hi there", // kept verbatim
	}
	got := cli.RedactParams(in)
	for _, secret := range []string{"mail_password", "auth_data", "new_password", "api_token", "db_secret", "ftp_passwort", "db_passwort"} {
		if got[secret] != "<redacted>" {
			t.Errorf("RedactParams[%q] = %q, want <redacted>", secret, got[secret])
		}
	}
	if got["mail_login"] != "m0000001" {
		t.Errorf("RedactParams[mail_login] = %q, want m0000001", got["mail_login"])
	}
	if got["record_id"] != "42" {
		t.Errorf("RedactParams[record_id] = %q, want \"42\"", got["record_id"])
	}
	if got["comment"] != "hi there" {
		t.Errorf("RedactParams[comment] = %q, want \"hi there\"", got["comment"])
	}
	// No secret value may survive anywhere in the rendered map.
	for k, v := range got {
		if strings.Contains(v, "hunter2") || strings.Contains(v, "topsecret") || strings.Contains(v, "abc123") ||
			strings.Contains(v, "geheim") {
			t.Errorf("secret leaked via %q = %q", k, v)
		}
	}
	if cli.RedactParams(nil) != nil {
		t.Errorf("RedactParams(nil) = non-nil, want nil")
	}
}

// Multi-line or oversized parameter values (the wholesale mailing-list
// config / subscriber blobs of update_mailinglist) must never reach the
// audit sinks verbatim: the list config can carry the list password in
// cleartext.
func TestRedactParamsElidesBlobs(t *testing.T) {
	t.Parallel()
	got := cli.RedactParams(map[string]any{
		"config":     "line1\npassword secret123\n",
		"subscriber": "a@x.de\rb@x.de",
		"long":       strings.Repeat("x", 300),
		"comment":    "short stays",
	})
	if got["config"] != "<elided 25 bytes>" {
		t.Errorf("config = %q, want <elided 25 bytes>", got["config"])
	}
	if got["subscriber"] != "<elided 13 bytes>" {
		t.Errorf("subscriber = %q, want <elided 13 bytes>", got["subscriber"])
	}
	if got["long"] != "<elided 300 bytes>" {
		t.Errorf("long = %q, want <elided 300 bytes>", got["long"])
	}
	if got["comment"] != "short stays" {
		t.Errorf("comment = %q, want kept verbatim", got["comment"])
	}
	for k, v := range got {
		if strings.Contains(v, "secret123") {
			t.Errorf("blob content leaked via %q = %q", k, v)
		}
	}
}

func TestOutcomeFor(t *testing.T) {
	t.Parallel()
	if got := cli.OutcomeFor(nil); got != "success" {
		t.Errorf("OutcomeFor(nil) = %q, want success", got)
	}
	var faulted error = &api.Error{Code: "mailaccount_not_found"}
	if got := cli.OutcomeFor(faulted); got != "failure:mailaccount_not_found" {
		t.Errorf("OutcomeFor(api fault) = %q, want failure:mailaccount_not_found", got)
	}
	if got := cli.OutcomeFor(errTransport{}); got != "failure" {
		t.Errorf("OutcomeFor(transport err) = %q, want failure", got)
	}
}

type errTransport struct{}

func (errTransport) Error() string { return "dial tcp: connection refused" }

func TestAuditRecordLogfmt(t *testing.T) {
	t.Parallel()
	var stderr bytes.Buffer
	rec := cli.AuditRecord{
		Time:    time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC),
		Login:   "w0000001",
		Action:  "delete_dns_settings",
		Target:  "record 42 in zone a b", // forces quoting
		Outcome: "success",
		Fields:  cli.RedactParams(map[string]any{"record_id": "42", "mail_password": "hunter2"}),
	}
	if err := cli.WriteAudit(&stderr, nil, rec); err != nil {
		t.Fatalf("WriteAudit: %v", err)
	}
	line := stderr.String()
	for _, want := range []string{
		"ts=2026-05-16T12:00:00Z",
		"login=w0000001",
		"action=delete_dns_settings",
		`target="record 42 in zone a b"`,
		"outcome=success",
		"record_id=42",
		"mail_password=<redacted>",
	} {
		if !strings.Contains(line, want) {
			t.Errorf("logfmt missing %q in:\n%s", want, line)
		}
	}
	if strings.Contains(line, "hunter2") {
		t.Errorf("secret leaked to stderr: %s", line)
	}
	if !strings.HasSuffix(line, "\n") {
		t.Errorf("logfmt line not newline-terminated: %q", line)
	}
}

// A field value containing a newline must not split the stderr audit
// record across physical lines: the embedded newline is escaped to the
// two-character \n inside a quoted value, so the record stays a single
// logfmt line. RedactParams elides multi-line blobs before they reach
// Fields, so the map is built directly here — the escaping is
// defense-in-depth for values arriving through another path.
func TestAuditRecordLogfmtEscapesNewlines(t *testing.T) {
	t.Parallel()
	var stderr bytes.Buffer
	rec := cli.AuditRecord{
		Time:    time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC),
		Login:   "w0000001",
		Action:  "update_mailinglist",
		Target:  "announce-example-com",
		Outcome: "success",
		Fields:  map[string]string{"subscriber": "a@x.de\nb@x.de"},
	}
	if err := cli.WriteAudit(&stderr, nil, rec); err != nil {
		t.Fatalf("WriteAudit: %v", err)
	}
	line := stderr.String()
	if got := strings.Count(line, "\n"); got != 1 {
		t.Errorf("record spans %d newlines, want exactly 1 (the terminator):\n%q", got, line)
	}
	if !strings.Contains(line, `subscriber="a@x.de\nb@x.de"`) {
		t.Errorf("newline not escaped to \\n in:\n%q", line)
	}
}

func TestWriteAuditFileJSONL(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")
	f, err := cli.OpenAuditFile(path)
	if err != nil {
		t.Fatalf("OpenAuditFile: %v", err)
	}
	var stderr bytes.Buffer
	mk := func(action string) cli.AuditRecord {
		return cli.AuditRecord{
			Time:    time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC),
			Login:   "w0000001",
			Action:  action,
			Target:  "m0000001",
			Outcome: "success",
			Fields:  cli.RedactParams(map[string]any{"mail_password": "hunter2"}),
		}
	}
	if err = cli.WriteAudit(&stderr, f, mk("add_mailaccount")); err != nil {
		t.Fatalf("WriteAudit 1: %v", err)
	}
	if err = cli.WriteAudit(&stderr, f, mk("delete_mailaccount")); err != nil {
		t.Fatalf("WriteAudit 2: %v", err)
	}
	if err = f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("audit file mode = %o, want 600", perm)
	}

	//nolint:gosec // G304: path is a test-controlled t.TempDir() file.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if strings.Contains(string(data), "hunter2") {
		t.Errorf("secret leaked to file:\n%s", data)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("file has %d JSON lines, want 2:\n%s", len(lines), data)
	}
	var first cli.AuditRecord
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("line 0 not valid JSON: %v", err)
	}
	if first.Action != "add_mailaccount" || first.Fields["mail_password"] != "<redacted>" {
		t.Errorf("decoded record 0 = %+v", first)
	}
}

func TestAuditLogPathPrecedence(t *testing.T) {
	t.Setenv("KAS_AUDIT_LOG", "/env/audit.log")
	if got := cli.AuditLogPath(&cli.RootOptions{AuditLog: "/flag/audit.log"}); got != "/flag/audit.log" {
		t.Errorf("flag should win: got %q", got)
	}
	if got := cli.AuditLogPath(&cli.RootOptions{}); got != "/env/audit.log" {
		t.Errorf("env fallback: got %q, want /env/audit.log", got)
	}
	t.Setenv("KAS_AUDIT_LOG", "")
	if got := cli.AuditLogPath(&cli.RootOptions{}); got != "" {
		t.Errorf("no flag, no env: got %q, want \"\"", got)
	}
	if got := cli.AuditLogPath(nil); got != "" {
		t.Errorf("nil opts, no env: got %q, want \"\"", got)
	}
}
