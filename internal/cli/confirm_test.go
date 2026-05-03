package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/cli"
)

func TestConfirm(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  bool
	}{
		{"y\n", true},
		{"Y\n", true},
		{"yes\n", true},
		{"YES\n", true},
		{"  yes  \n", true},
		{"n\n", false},
		{"no\n", false},
		{"\n", false},
		{"maybe\n", false},
		{"", false},
	}
	for _, tt := range tests {
		var out bytes.Buffer
		got, err := cli.Confirm(strings.NewReader(tt.input), &out, "delete?")
		if err != nil {
			t.Errorf("Confirm(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("Confirm(%q) = %v, want %v", tt.input, got, tt.want)
		}
		if !strings.Contains(out.String(), "delete? [y/N]:") {
			t.Errorf("Confirm(%q) prompt missing: %q", tt.input, out.String())
		}
	}
}
