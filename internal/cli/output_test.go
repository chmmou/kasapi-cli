package cli_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/cli"
)

type sample struct {
	Name  string `json:"name" yaml:"name"`
	Count int    `json:"count" yaml:"count"`
}

type sampleTable struct {
	rows [][]string
}

func (sampleTable) TableHeaders() []string  { return []string{"Name", "Count"} }
func (s sampleTable) TableRows() [][]string { return s.rows }

func TestParseFormat(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in      string
		want    cli.Format
		wantErr bool
	}{
		{"", cli.DefaultFormat, false},
		{"json", cli.FormatJSON, false},
		{"yaml", cli.FormatYAML, false},
		{"table", cli.FormatTable, false},
		{"JSON", "", true},
		{"xml", "", true},
	}
	for _, tt := range tests {
		got, err := cli.ParseFormat(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseFormat(%q): want error, got %q", tt.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseFormat(%q): unexpected error: %v", tt.in, err)
		}
		if got != tt.want {
			t.Errorf("ParseFormat(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRenderJSON(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	if err := cli.Render(&buf, cli.FormatJSON, sample{Name: "alpha", Count: 7}); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.Contains(got, `"name": "alpha"`) || !strings.Contains(got, `"count": 7`) {
		t.Errorf("json output missing fields: %q", got)
	}
}

func TestRenderYAML(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	if err := cli.Render(&buf, cli.FormatYAML, sample{Name: "beta", Count: 9}); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.Contains(got, "name: beta") || !strings.Contains(got, "count: 9") {
		t.Errorf("yaml output missing fields: %q", got)
	}
}

func TestRenderTableTabular(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	tbl := sampleTable{rows: [][]string{{"a", "1"}, {"bb", "22"}}}
	if err := cli.Render(&buf, cli.FormatTable, tbl); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	for _, want := range []string{"Name", "Count", "a", "1", "bb", "22"} {
		if !strings.Contains(got, want) {
			t.Errorf("table output missing %q: %q", want, got)
		}
	}
}

func TestRenderTableNonTabularReturnsError(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := cli.Render(&buf, cli.FormatTable, sample{Name: "x", Count: 1})
	if !errors.Is(err, cli.ErrTableNotSupported) {
		t.Fatalf("want ErrTableNotSupported, got %v", err)
	}
}
