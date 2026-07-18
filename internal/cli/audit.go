package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/chmmou/kasapi-cli/internal/api"
)

// AuditRecord is the structured trace emitted for one dispatched write
// action. It is written after the SOAP call returns, regardless of
// outcome — a failed write is at least as important to log as a
// successful one. Fields holds already-redacted correlating values
// (e.g. record_id from add_dns_settings); secret request parameters
// must be filtered with RedactParams before they reach this struct.
type AuditRecord struct {
	Time    time.Time         `json:"time"`
	Login   string            `json:"login"`
	Action  string            `json:"action"`
	Target  string            `json:"target"`
	Outcome string            `json:"outcome"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// AuditOutcomeDryRun is the AuditRecord.Outcome value written for a
// --dry-run invocation (#132): the request shape was previewed but no
// SOAP call was dispatched, so it must be distinguishable in the log
// from a real success or failure.
const AuditOutcomeDryRun = "dry-run"

// OutcomeFor maps a write call's error to the audit outcome string:
// "success" on nil, "failure:<kas_code>" for a typed KAS fault, and a
// bare "failure" for any other (transport/decode) error.
func OutcomeFor(err error) string {
	if err == nil {
		return "success"
	}
	if e := api.AsError(err); e != nil {
		return "failure:" + e.Code
	}
	return "failure"
}

// auditSecretParams are request-parameter keys whose values must never
// reach the audit log. The list is intentionally conservative and grows
// as the #13 write endpoints land; the substring rule in redactParam
// additionally catches the common *password / *_token / *secret shapes
// so a newly wired endpoint is redaction-safe by default.
var auditSecretParams = map[string]struct{}{
	"auth_data":         {},
	"kas_auth_data":     {},
	"password":          {},
	"new_password":      {},
	"mail_password":     {},
	"database_password": {},
	"ftp_password":      {},
	"samba_password":    {},
	"session":           {},
	"token":             {},
}

const auditRedacted = "<redacted>"

// maxAuditValueLen caps how long a single parameter value may be before
// RedactParams elides it. Normal write parameters (names, hosts, Y/N
// toggles) are far shorter; only wholesale blobs like the mailing-list
// config exceed it.
const maxAuditValueLen = 256

// redactParam reports whether the value of parameter key must be
// redacted before it is logged.
func redactParam(key string) bool {
	k := strings.ToLower(key)
	if _, ok := auditSecretParams[k]; ok {
		return true
	}
	// "passwort" is the German spelling the KAS API uses for several
	// keys (ftp_passwort, db_passwort, …); it must be caught alongside
	// the English "password" so a future write slice sending one is
	// redaction-safe by default.
	for _, frag := range []string{"password", "passwort", "passwd", "secret", "token", "auth_data"} {
		if strings.Contains(k, frag) {
			return true
		}
	}
	return false
}

// RedactParams converts a KAS request/response parameter map into the
// string map stored on AuditRecord.Fields, replacing every secret value
// (see redactParam) with auditRedacted. Multi-line or oversized values
// (mailing-list config / subscriber blobs sent wholesale by
// update_mailinglist) are elided to a "<elided N bytes>" marker: the
// list config can carry the list password in cleartext, so the blob
// content must never reach either audit sink. Non-string values are
// rendered with %v. A nil/empty map yields nil so the field is omitted.
func RedactParams(params map[string]any) map[string]string {
	if len(params) == 0 {
		return nil
	}
	out := make(map[string]string, len(params))
	for k, v := range params {
		if redactParam(k) {
			out[k] = auditRedacted
			continue
		}
		s := fmt.Sprintf("%v", v)
		if strings.ContainsAny(s, "\n\r") || len(s) > maxAuditValueLen {
			s = fmt.Sprintf("<elided %d bytes>", len(s))
		}
		out[k] = s
	}
	return out
}

// logfmt renders the record as a single grep-friendly key=value line
// without a trailing newline. Field keys are emitted in sorted order so
// the output is deterministic.
func (r AuditRecord) logfmt() string {
	var b strings.Builder
	b.WriteString("ts=")
	b.WriteString(r.Time.UTC().Format(time.RFC3339Nano))
	b.WriteString(" login=")
	b.WriteString(quoteIfNeeded(r.Login))
	b.WriteString(" action=")
	b.WriteString(quoteIfNeeded(r.Action))
	b.WriteString(" target=")
	b.WriteString(quoteIfNeeded(r.Target))
	b.WriteString(" outcome=")
	b.WriteString(quoteIfNeeded(r.Outcome))
	keys := make([]string, 0, len(r.Fields))
	for k := range r.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.WriteByte(' ')
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(quoteIfNeeded(r.Fields[k]))
	}
	return b.String()
}

// quoteIfNeeded wraps v in double quotes when it is empty or contains
// whitespace, a quote, or '=' so the logfmt line stays unambiguous to
// split on. Backslash and quote are escaped; a newline or carriage
// return is escaped to the two-character \n / \r so a single field
// value can never split the record across physical lines. RedactParams
// already elides multi-line blobs, so this escaping is defense-in-depth
// for values that reach Fields through another path.
func quoteIfNeeded(v string) string {
	if v == "" {
		return `""`
	}
	if strings.ContainsAny(v, " \t\r\n\"=") {
		return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\r", `\r`).Replace(v) + `"`
	}
	return v
}

// OpenAuditFile opens path for appending JSON-Lines audit records,
// creating it with mode 0600. As in the session store, the create-mode
// bits are advisory on Windows, so an explicit Chmod re-asserts 0600 on
// an already-existing file.
func OpenAuditFile(path string) (*os.File, error) {
	//nolint:gosec // G304: the audit-log path is an explicit user-supplied --audit-log / KAS_AUDIT_LOG value; opening exactly that file is the feature.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("cli: open audit log %s: %w", path, err)
	}
	if cherr := f.Chmod(0o600); cherr != nil {
		_ = f.Close()
		return nil, fmt.Errorf("cli: chmod audit log %s: %w", path, cherr)
	}
	return f, nil
}

// WriteAudit emits r to stderr as a logfmt line (always, independent of
// --verbose) and, when file is non-nil, appends it as one JSON object
// (JSON Lines) to file. The stderr line is written first so the trace
// survives a broken --audit-log path. Secrets must already have been
// removed from r.Fields via RedactParams.
func WriteAudit(stderr io.Writer, file io.Writer, r AuditRecord) error {
	if _, err := fmt.Fprintln(stderr, r.logfmt()); err != nil {
		return fmt.Errorf("cli: write audit stderr: %w", err)
	}
	if file == nil {
		return nil
	}
	line, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("cli: marshal audit record: %w", err)
	}
	if _, err := file.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("cli: write audit file: %w", err)
	}
	return nil
}

// AuditLogPath resolves the audit-log file path: the --audit-log flag
// wins, falling back to the KAS_AUDIT_LOG environment variable; ""
// means stderr-only.
func AuditLogPath(opts *RootOptions) string {
	if opts != nil && opts.AuditLog != "" {
		return opts.AuditLog
	}
	return os.Getenv("KAS_AUDIT_LOG")
}
