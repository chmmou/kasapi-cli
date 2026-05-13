package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/goccy/go-yaml"
)

// Format identifies an output renderer chosen via --output.
type Format string

// Output formats accepted by --output. AllFormats is the canonical
// ordered listing; DefaultFormat is what an unset --output resolves to.
const (
	FormatJSON  Format = "json"
	FormatYAML  Format = "yaml"
	FormatTable Format = "table"
)

// DefaultFormat is what --output defaults to when the flag is unset.
const DefaultFormat = FormatTable

// AllFormats is the canonical, ordered list of accepted --output values.
// Used by the root command help text and validation.
var AllFormats = []Format{FormatJSON, FormatYAML, FormatTable}

// formatNames is the cached []string view of AllFormats. Single source
// of truth for the strings used by both the --output flag help (joined
// with "|") and the ParseFormat error message (joined with ", "); built
// once at package init so callers do not re-allocate per call. Treat as
// read-only.
var formatNames = func() []string {
	names := make([]string, len(AllFormats))
	for i, f := range AllFormats {
		names[i] = string(f)
	}
	return names
}()

// ParseFormat returns the Format matching s or an error listing valid
// values. The empty string maps to DefaultFormat.
func ParseFormat(s string) (Format, error) {
	if s == "" {
		return DefaultFormat, nil
	}
	for _, f := range AllFormats {
		if string(f) == s {
			return f, nil
		}
	}
	return "", fmt.Errorf("invalid output format %q (want one of %s)", s, strings.Join(formatNames, ", "))
}

// Tabular is implemented by values that can render themselves as a
// fixed-column table. Subcommand result types satisfy it so --output=table
// works without per-renderer special cases.
type Tabular interface {
	TableHeaders() []string
	TableRows() [][]string
}

// Render writes v to w in the requested format. For FormatTable the
// value must implement Tabular; otherwise Render returns an error so
// callers can surface it as a user error.
func Render(w io.Writer, format Format, v any) error {
	switch format {
	case FormatJSON:
		return renderJSON(w, v)
	case FormatYAML:
		return renderYAML(w, v)
	case FormatTable:
		return renderTable(w, v)
	default:
		return fmt.Errorf("cli: unknown output format %q", format)
	}
}

func renderJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func renderYAML(w io.Writer, v any) error {
	out, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	_, err = w.Write(out)
	return err
}

// ErrTableNotSupported is returned by Render when --output=table is
// requested for a value that does not implement Tabular. Every
// subcommand result type is expected to implement Tabular, so reaching
// this error means a subcommand is missing its TableHeaders/TableRows
// pair — i.e. a kasapi-cli bug, not a user choice. The workaround hint
// keeps the user unblocked in the meantime.
var ErrTableNotSupported = errors.New("table output not implemented for this command (programming bug; use --output=json or --output=yaml)")

func renderTable(w io.Writer, v any) error {
	t, ok := v.(Tabular)
	if !ok {
		return ErrTableNotSupported
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	headers := t.TableHeaders()
	if len(headers) > 0 {
		if _, err := fmt.Fprintln(tw, strings.Join(headers, "\t")); err != nil {
			return err
		}
	}
	for _, row := range t.TableRows() {
		if _, err := fmt.Fprintln(tw, strings.Join(row, "\t")); err != nil {
			return err
		}
	}
	return tw.Flush()
}
