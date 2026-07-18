package cli

import (
	"io"
	"sort"
	"time"
)

// DryRunPreview is the structured description of the KAS request a
// destructive command would have dispatched, shown to the user under
// --dry-run instead of contacting the API. Params is already redacted
// (see RedactParams); rendering honours --output (table/json/yaml).
type DryRunPreview struct {
	Action string            `json:"action" yaml:"action"`
	Target string            `json:"target" yaml:"target"`
	Params map[string]string `json:"params,omitempty" yaml:"params,omitempty"`
}

// NewDryRunPreview builds a preview for the given KAS action and target
// resource, redacting secret request parameters with RedactParams so a
// password or auth token never reaches stdout or a piped log.
func NewDryRunPreview(kasAction, target string, params map[string]any) DryRunPreview {
	return DryRunPreview{
		Action: kasAction,
		Target: target,
		Params: RedactParams(params),
	}
}

// TableHeaders implements Tabular for --output=table.
func (DryRunPreview) TableHeaders() []string { return []string{"FIELD", "VALUE"} }

// TableRows implements Tabular: the action and target first, then one
// row per redacted parameter (sorted) as "param.<key>".
func (p DryRunPreview) TableRows() [][]string {
	rows := [][]string{
		{"action", p.Action},
		{"target", p.Target},
	}
	keys := make([]string, 0, len(p.Params))
	for k := range p.Params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		rows = append(rows, []string{"param." + k, p.Params[k]})
	}
	return rows
}

// WriteResolver is the common shape of ResolveDestructive and
// ResolveWrite so a write runner can hold either behind one variable
// and dispatch uniformly. ResolveWrite ignores in/isTTY (no prompt);
// the parameters stay for signature parity.
type WriteResolver func(
	opts *RootOptions,
	in io.Reader,
	out, stderr, auditFile io.Writer,
	isTTY bool,
	login, kasAction string,
	confirm ConfirmAction,
	params map[string]any,
) (proceed bool, err error)

// previewAndAudit runs the shared --dry-run short-circuit (#132 + #131)
// every write consults: when opts.DryRun is set it renders the request
// to out in opts.Output and writes an audit record with outcome
// "dry-run", reporting handled=true so the caller returns without
// dispatching. When opts.DryRun is unset it reports handled=false and
// the caller proceeds to its own gate / dispatch.
func previewAndAudit(
	opts *RootOptions,
	out, stderr, auditFile io.Writer,
	login, kasAction string,
	confirm ConfirmAction,
	params map[string]any,
) (handled bool, err error) {
	if opts == nil || !opts.DryRun {
		return false, nil
	}
	preview := NewDryRunPreview(kasAction, confirm.ID, params)
	if rerr := Render(out, opts.Output, preview); rerr != nil {
		return true, UserError(rerr, "dry-run preview")
	}
	rec := AuditRecord{
		Time:    time.Now().UTC(),
		Login:   login,
		Action:  kasAction,
		Target:  confirm.ID,
		Outcome: AuditOutcomeDryRun,
		Fields:  preview.Params,
	}
	if aerr := WriteAudit(stderr, auditFile, rec); aerr != nil {
		return true, UserError(aerr, "dry-run audit")
	}
	return true, nil
}

// ResolveDestructive is the shared dispatch gate every *destructive*
// command consults before its SOAP call. It composes the --dry-run
// preview (#132), the confirmation gate (#109) and the audit log
// (#131):
//
//   - opts.DryRun: render the request to out in opts.Output, write an
//     audit record with outcome "dry-run", and return (false, nil). No
//     prompt and no dispatch happen — even when --yes is also set.
//   - otherwise: run GateDestructive. A declined or non-interactive
//     confirmation returns (false, err); a pass returns (true, nil) and
//     the caller dispatches, then emits its own success/failure audit
//     record via OutcomeFor.
//
// login and kasAction plus confirm.ID identify the target for the
// preview and the audit trace. auditFile is the optional --audit-log
// sink (nil = stderr only).
func ResolveDestructive(
	opts *RootOptions,
	in io.Reader,
	out, stderr, auditFile io.Writer,
	isTTY bool,
	login, kasAction string,
	confirm ConfirmAction,
	params map[string]any,
) (proceed bool, err error) {
	if handled, herr := previewAndAudit(opts, out, stderr, auditFile, login, kasAction, confirm, params); handled {
		return false, herr
	}
	// The prompt goes to stderr, not out: stdout carries the command's
	// machine-readable result, and a redirected `cmd > file` must not
	// swallow the [y/N] question the user is expected to answer.
	if gerr := GateDestructive(in, stderr, isTTY, opts != nil && opts.Yes, confirm); gerr != nil {
		return false, gerr
	}
	return true, nil
}

// ResolveWrite is the non-destructive counterpart of
// ResolveDestructive for write actions that mutate but are reversible
// (e.g. add_mailforward): the #131 audit log and #132 --dry-run still
// apply to every write, but no #109 confirmation prompt is shown.
// Under --dry-run it behaves exactly like ResolveDestructive
// (preview + dry-run audit, no dispatch); otherwise it returns
// (true, nil) for the caller to dispatch and audit. in and isTTY are
// unused and present only for WriteResolver signature parity.
func ResolveWrite(
	opts *RootOptions,
	_ io.Reader,
	out, stderr, auditFile io.Writer,
	_ bool,
	login, kasAction string,
	confirm ConfirmAction,
	params map[string]any,
) (proceed bool, err error) {
	if handled, herr := previewAndAudit(opts, out, stderr, auditFile, login, kasAction, confirm, params); handled {
		return false, herr
	}
	return true, nil
}
