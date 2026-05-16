package cli_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/cli"
)

func TestDryRunPreviewRedactsAndRenders(t *testing.T) {
	t.Parallel()
	p := cli.NewDryRunPreview("add_mailaccount", "m0000001", map[string]any{
		"mail_login":    "m0000001",
		"mail_password": "hunter2",
	})
	for _, format := range []cli.Format{cli.FormatTable, cli.FormatJSON, cli.FormatYAML} {
		var buf bytes.Buffer
		if err := cli.Render(&buf, format, p); err != nil {
			t.Fatalf("Render(%s): %v", format, err)
		}
		out := buf.String()
		if !strings.Contains(out, "add_mailaccount") || !strings.Contains(out, "m0000001") {
			t.Errorf("%s output missing action/target:\n%s", format, out)
		}
		// The mail_password key must survive (redacted, not dropped) and
		// its secret value must never appear, in any format. The exact
		// "<redacted>" marker is asserted format-free in TestRedactParams;
		// the JSON encoder additionally HTML-escapes the angle brackets,
		// so do not string-match the marker here.
		if !strings.Contains(out, "mail_password") {
			t.Errorf("%s output dropped the mail_password key:\n%s", format, out)
		}
		if strings.Contains(out, "hunter2") {
			t.Errorf("%s output leaked the password:\n%s", format, out)
		}
	}
}

func TestResolveDestructiveDryRunShortCircuits(t *testing.T) {
	t.Parallel()
	confirm := cli.ConfirmAction{Verb: "delete", Resource: "mail account", ID: "m0000001"}
	params := map[string]any{"mail_login": "m0000001", "mail_password": "hunter2"}

	cases := []struct {
		name  string
		yes   bool
		isTTY bool
	}{
		{"interactive, no --yes", false, true},
		{"with --yes", true, false},
		{"non-tty, no --yes", false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			opts := &cli.RootOptions{DryRun: true, Output: cli.FormatJSON, Yes: c.yes}
			var out, stderr, auditFile bytes.Buffer
			// The reader provides "y\n" but must never be consumed: dry-run
			// short-circuits the prompt entirely.
			proceed, err := cli.ResolveDestructive(
				opts, strings.NewReader("y\n"), &out, &stderr, &auditFile,
				c.isTTY, "w0000001", "delete_mailaccount", confirm, params)
			if proceed || err != nil {
				t.Fatalf("proceed=%v err=%v, want false/nil (no dispatch)", proceed, err)
			}
			if strings.Contains(out.String(), "[y/N]") {
				t.Errorf("dry-run prompted; out=%q", out.String())
			}
			var preview cli.DryRunPreview
			if jerr := json.Unmarshal(out.Bytes(), &preview); jerr != nil {
				t.Fatalf("preview not valid JSON: %v\n%s", jerr, out.String())
			}
			if preview.Action != "delete_mailaccount" || preview.Target != "m0000001" {
				t.Errorf("preview = %+v", preview)
			}
			if preview.Params["mail_password"] != "<redacted>" {
				t.Errorf("password not redacted in preview: %+v", preview.Params)
			}
			if !strings.Contains(stderr.String(), "outcome=dry-run") {
				t.Errorf("audit stderr missing dry-run marker: %q", stderr.String())
			}
			if strings.Contains(out.String()+stderr.String()+auditFile.String(), "hunter2") {
				t.Errorf("password leaked somewhere; out=%q stderr=%q file=%q",
					out.String(), stderr.String(), auditFile.String())
			}
			var rec cli.AuditRecord
			if jerr := json.Unmarshal(bytes.TrimSpace(auditFile.Bytes()), &rec); jerr != nil {
				t.Fatalf("audit file line not JSON: %v\n%s", jerr, auditFile.String())
			}
			if rec.Outcome != cli.AuditOutcomeDryRun {
				t.Errorf("audit outcome = %q, want %q", rec.Outcome, cli.AuditOutcomeDryRun)
			}
		})
	}
}

func TestResolveDestructiveDelegatesToGate(t *testing.T) {
	t.Parallel()
	confirm := cli.ConfirmAction{Verb: "delete", Resource: "mail account", ID: "m0000001"}

	t.Run("declined", func(t *testing.T) {
		t.Parallel()
		opts := &cli.RootOptions{DryRun: false}
		var out, stderr bytes.Buffer
		proceed, err := cli.ResolveDestructive(
			opts, strings.NewReader("n\n"), &out, &stderr, nil,
			true, "w0000001", "delete_mailaccount", confirm, nil)
		if proceed || !errors.Is(err, cli.ErrConfirmationDeclined) {
			t.Fatalf("proceed=%v err=%v, want false/ErrConfirmationDeclined", proceed, err)
		}
		if !strings.Contains(out.String(), "[y/N]") {
			t.Errorf("expected a prompt; out=%q", out.String())
		}
	})

	t.Run("accepted", func(t *testing.T) {
		t.Parallel()
		opts := &cli.RootOptions{DryRun: false}
		var out, stderr bytes.Buffer
		proceed, err := cli.ResolveDestructive(
			opts, strings.NewReader("y\n"), &out, &stderr, nil,
			true, "w0000001", "delete_mailaccount", confirm, nil)
		if !proceed || err != nil {
			t.Fatalf("proceed=%v err=%v, want true/nil", proceed, err)
		}
	})

	t.Run("yes bypasses prompt", func(t *testing.T) {
		t.Parallel()
		opts := &cli.RootOptions{DryRun: false, Yes: true}
		var out, stderr bytes.Buffer
		proceed, err := cli.ResolveDestructive(
			opts, strings.NewReader(""), &out, &stderr, nil,
			false, "w0000001", "delete_mailaccount", confirm, nil)
		if !proceed || err != nil {
			t.Fatalf("proceed=%v err=%v, want true/nil", proceed, err)
		}
		if strings.Contains(out.String(), "[y/N]") {
			t.Errorf("--yes should not prompt; out=%q", out.String())
		}
	})

	t.Run("non-tty without yes", func(t *testing.T) {
		t.Parallel()
		opts := &cli.RootOptions{DryRun: false}
		var out, stderr bytes.Buffer
		proceed, err := cli.ResolveDestructive(
			opts, strings.NewReader(""), &out, &stderr, nil,
			false, "w0000001", "delete_mailaccount", confirm, nil)
		if proceed || !errors.Is(err, cli.ErrConfirmationRequired) {
			t.Fatalf("proceed=%v err=%v, want false/ErrConfirmationRequired", proceed, err)
		}
	})
}
