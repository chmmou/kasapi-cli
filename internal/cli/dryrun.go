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

// ResolveDestructive is the shared dispatch gate every destructive
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
	if opts != nil && opts.DryRun {
		preview := NewDryRunPreview(kasAction, confirm.ID, params)
		if rerr := Render(out, opts.Output, preview); rerr != nil {
			return false, UserError(rerr, "dry-run preview")
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
			return false, UserError(aerr, "dry-run audit")
		}
		return false, nil
	}
	if gerr := GateDestructive(in, out, isTTY, opts != nil && opts.Yes, confirm); gerr != nil {
		return false, gerr
	}
	return true, nil
}
