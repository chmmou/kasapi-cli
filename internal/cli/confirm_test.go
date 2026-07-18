package cli_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"testing/iotest"

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

func TestGateDestructive(t *testing.T) {
	t.Parallel()
	action := cli.ConfirmAction{Verb: "delete", Resource: "mail account", ID: "m0000001"}

	tests := []struct {
		name       string
		input      string
		isTTY      bool
		yes        bool
		wantErr    error // nil = proceed
		wantPrompt bool  // whether the [y/N] summary must appear on out
	}{
		{name: "accepted", input: "y\n", isTTY: true, yes: false, wantErr: nil, wantPrompt: true},
		{name: "declined", input: "n\n", isTTY: true, yes: false, wantErr: cli.ErrConfirmationDeclined, wantPrompt: true},
		{name: "empty declines", input: "\n", isTTY: true, yes: false, wantErr: cli.ErrConfirmationDeclined, wantPrompt: true},
		{name: "yes bypasses prompt", input: "", isTTY: false, yes: true, wantErr: nil, wantPrompt: false},
		{name: "non-tty without yes", input: "", isTTY: false, yes: false, wantErr: cli.ErrConfirmationRequired, wantPrompt: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			err := cli.GateDestructive(strings.NewReader(tt.input), &out, tt.isTTY, tt.yes, action)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("GateDestructive() = %v, want nil (proceed)", err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("GateDestructive() = %v, want wrapped %v", err, tt.wantErr)
				}
				if got := cli.CodeFor(err); got != cli.ExitUserError {
					t.Errorf("CodeFor = %d, want ExitUserError (%d)", got, cli.ExitUserError)
				}
			}
			hasPrompt := strings.Contains(out.String(), `About to delete mail account "m0000001". This cannot be undone.`)
			if hasPrompt != tt.wantPrompt {
				t.Errorf("prompt shown = %v, want %v; out=%q", hasPrompt, tt.wantPrompt, out.String())
			}
		})
	}
}

// A prompt I/O failure (neither a yes nor a no was read) is an aborted
// attempt: ErrConfirmationAborted, exit 1, so the write runner can
// audit it as outcome=aborted.
func TestGateDestructiveAbortsOnPromptIOError(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	action := cli.ConfirmAction{Verb: "delete", Resource: "mail account", ID: "m0000001"}
	err := cli.GateDestructive(iotest.ErrReader(errors.New("tty gone")), &out, true, false, action)
	if !errors.Is(err, cli.ErrConfirmationAborted) {
		t.Errorf("err = %v, want wrapped ErrConfirmationAborted", err)
	}
	if got := cli.CodeFor(err); got != cli.ExitUserError {
		t.Errorf("CodeFor = %d, want ExitUserError (%d)", got, cli.ExitUserError)
	}
}
