package cli_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/cli"
)

func TestMailCmdHelpListsSubcommandGroups(t *testing.T) {
	t.Parallel()
	root, opts := cli.NewRootCmd()
	root.AddCommand(cli.NewMailCmd(opts))

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"mail", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"accounts", "forwards", "filters", "lists"} {
		if !strings.Contains(out, want) {
			t.Errorf("--help output missing %q\n%s", want, out)
		}
	}
}

func TestMailAccountsHelpListsSubcommands(t *testing.T) {
	t.Parallel()
	root, opts := cli.NewRootCmd()
	root.AddCommand(cli.NewMailCmd(opts))

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"mail", "accounts", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"list", "get", "add", "update", "delete"} {
		if !strings.Contains(out, want) {
			t.Errorf("--help output missing %q\n%s", want, out)
		}
	}
}

func TestMailAccountsAddRejectsBadInput(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		args []string
	}{
		{"address without @", []string{"mail", "accounts", "add", "notanemail", "--password", "pw"}},
		{"missing --password", []string{"mail", "accounts", "add", "info@example.com"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			root, opts := cli.NewRootCmd()
			root.AddCommand(cli.NewMailCmd(opts))
			var buf bytes.Buffer
			root.SetOut(&buf)
			root.SetErr(&buf)
			root.SetArgs(c.args)
			err := root.Execute()
			if err == nil {
				t.Fatalf("Execute %v: want error, got nil", c.args)
			}
			if cli.CodeFor(err) != cli.ExitUserError {
				t.Errorf("exit code = %d, want ExitUserError", cli.CodeFor(err))
			}
		})
	}
}

// update_mailaccount with no field flags is rejected at build time
// (before any gating or credential resolution) with a user error — the
// API would otherwise fault nothing_to_do. Mirrors the add input-
// rejection test.
func TestMailAccountsUpdateRejectsNoFields(t *testing.T) {
	t.Parallel()
	root, opts := cli.NewRootCmd()
	root.AddCommand(cli.NewMailCmd(opts))
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"mail", "accounts", "update", "m0000001"})
	err := root.Execute()
	if err == nil {
		t.Fatal("Execute: want error for update with no field flags, got nil")
	}
	if cli.CodeFor(err) != cli.ExitUserError {
		t.Errorf("exit code = %d, want ExitUserError", cli.CodeFor(err))
	}
}

// The destructive accounts subcommands (update/delete) must refuse on a
// non-interactive stdin without --yes rather than dispatch unconfirmed.
func TestMailAccountsDestructiveRefuseNonTTY(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"mail", "accounts", "delete", "m0000001"},
		{"mail", "accounts", "update", "m0000001", "--active", "N"},
	} {
		t.Run(args[2], func(t *testing.T) {
			t.Parallel()
			root, opts := cli.NewRootCmd()
			root.AddCommand(cli.NewMailCmd(opts))
			var buf bytes.Buffer
			root.SetOut(&buf)
			root.SetErr(&buf)
			root.SetIn(strings.NewReader(""))
			root.SetArgs(append(args,
				"--login", "w0000000", "--auth-data", "x", "--auth-type", "plain"))
			err := root.Execute()
			if err == nil {
				t.Fatalf("Execute %v: want refusal error, got nil", args)
			}
			if !errors.Is(err, cli.ErrConfirmationRequired) {
				t.Errorf("err = %v, want ErrConfirmationRequired", err)
			}
		})
	}
}

// delete_mailaccount uses the louder "permanently delete" verb (a
// deleted mailbox loses every stored message), shared only with
// delete_database. Pin the verb and the rendered Summary so neither a
// slice refactor nor a change to the shared ConfirmAction template can
// silently regress the loudness contract.
func TestMailAccountsDeleteConfirmIsLouder(t *testing.T) {
	t.Parallel()
	a := cli.MailAccountDeleteConfirm("m0000001")
	if a.Verb != "permanently delete" {
		t.Errorf("Verb = %q, want %q", a.Verb, "permanently delete")
	}
	if a.Resource != "mail account" {
		t.Errorf("Resource = %q, want %q", a.Resource, "mail account")
	}
	want := `About to permanently delete mail account "m0000001". This cannot be undone.`
	if got := a.Summary(); got != want {
		t.Errorf("Summary() = %q, want %q", got, want)
	}
}

// add_mailaccount is non-destructive (no prompt) but still routes
// through the dry-run/audit seam. The dry-run preview must split the
// address into local_part/domain_part, send the add-only mail_password,
// carry the documented default toggles, and never leak the update-only
// mail_login / mail_new_password / is_active keys.
func TestMailAccountsAddDryRunParams(t *testing.T) {
	t.Parallel()
	root, opts := cli.NewRootCmd()
	root.AddCommand(cli.NewMailCmd(opts))
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	root.SetArgs([]string{
		"mail", "accounts", "add", "info@example.com", "--password", "s3cret",
		"--dry-run", "-o", "json",
		"--login", "w0", "--auth-data", "x", "--auth-type", "plain",
	})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var got struct {
		Action string            `json:"action"`
		Params map[string]string `json:"params"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal preview: %v\nstdout:%s", err, out.String())
	}
	if got.Action != "add_mailaccount" {
		t.Errorf("action = %q, want add_mailaccount", got.Action)
	}
	want := map[string]string{
		"local_part":  "info",
		"domain_part": "example.com",
		// mail_password is a secret, so the audit/preview redacts its
		// value — the key is still present but the value is masked.
		"mail_password":      "<redacted>",
		"webmail_autologin":  "Y",
		"responder":          "N",
		"mail_xlist_enabled": "Y",
		"mail_xlist_sent":    "Sent",
	}
	for k, v := range want {
		if got.Params[k] != v {
			t.Errorf("params[%q] = %q, want %q (full: %v)", k, got.Params[k], v, got.Params)
		}
	}
	for _, k := range []string{"mail_login", "mail_new_password", "is_active"} {
		if _, ok := got.Params[k]; ok {
			t.Errorf("add params must not contain %q (full: %v)", k, got.Params)
		}
	}
}

// The accounts update field assembly mirrors the lists update: only the
// flags the user explicitly set are sent (keyed on cobra Changed), the
// password maps to the update-only mail_new_password key, and --active
// maps to is_active. --dry-run renders the exact KAS params, so this
// asserts the assembly end to end without a network call.
func TestMailAccountsUpdateDryRunFieldAssembly(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		args   []string
		want   map[string]string
		absent []string
	}{
		{
			"password maps to mail_new_password",
			[]string{"mail", "accounts", "update", "m0000001", "--password", "n3w"},
			// mail_new_password is a secret → redacted in the preview.
			map[string]string{"mail_login": "m0000001", "mail_new_password": "<redacted>"},
			[]string{"mail_password", "is_active", "responder"},
		},
		{
			"active and responder",
			[]string{"mail", "accounts", "update", "m0000001", "--active", "N", "--responder", "Y"},
			map[string]string{"mail_login": "m0000001", "is_active": "N", "responder": "Y"},
			[]string{"mail_new_password"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			root, opts := cli.NewRootCmd()
			root.AddCommand(cli.NewMailCmd(opts))
			var out, errb bytes.Buffer
			root.SetOut(&out)
			root.SetErr(&errb)
			args := append(append([]string{}, c.args...),
				"--dry-run", "-o", "json",
				"--login", "w0", "--auth-data", "x", "--auth-type", "plain")
			root.SetArgs(args)
			if err := root.Execute(); err != nil {
				t.Fatalf("Execute %v: %v", args, err)
			}
			var got struct {
				Action string            `json:"action"`
				Params map[string]string `json:"params"`
			}
			if err := json.Unmarshal(out.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal preview: %v\nstdout:%s", err, out.String())
			}
			if got.Action != "update_mailaccount" {
				t.Errorf("action = %q, want update_mailaccount", got.Action)
			}
			for k, v := range c.want {
				if got.Params[k] != v {
					t.Errorf("params[%q] = %q, want %q (full: %v)", k, got.Params[k], v, got.Params)
				}
			}
			for _, k := range c.absent {
				if _, ok := got.Params[k]; ok {
					t.Errorf("params[%q] present, want absent (full: %v)", k, got.Params)
				}
			}
		})
	}
}

func TestMailForwardsHelpListsSubcommands(t *testing.T) {
	t.Parallel()
	root, opts := cli.NewRootCmd()
	root.AddCommand(cli.NewMailCmd(opts))

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"mail", "forwards", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"list", "get", "add", "update", "delete"} {
		if !strings.Contains(out, want) {
			t.Errorf("--help output missing %q\n%s", want, out)
		}
	}
}

func TestMailForwardsAddRejectsBadInput(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		args []string
	}{
		{"address without @", []string{"mail", "forwards", "add", "notanemail", "--target", "a@b.de"}},
		{"missing --target", []string{"mail", "forwards", "add", "info@example.de"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			root, opts := cli.NewRootCmd()
			root.AddCommand(cli.NewMailCmd(opts))
			var buf bytes.Buffer
			root.SetOut(&buf)
			root.SetErr(&buf)
			root.SetArgs(c.args)
			err := root.Execute()
			if err == nil {
				t.Fatalf("Execute %v: want error, got nil", c.args)
			}
			if cli.CodeFor(err) != cli.ExitUserError {
				t.Errorf("exit code = %d, want ExitUserError", cli.CodeFor(err))
			}
		})
	}
}

// The destructive forwards subcommands (update/delete) must refuse on a
// non-interactive stdin without --yes rather than dispatch unconfirmed.
// SetIn with a non-terminal reader exercises the !isTTY path.
func TestMailForwardsDestructiveRefuseNonTTY(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"mail", "forwards", "delete", "info@example.de"},
		{"mail", "forwards", "update", "info@example.de", "--target", "a@b.de"},
	} {
		t.Run(args[2], func(t *testing.T) {
			t.Parallel()
			root, opts := cli.NewRootCmd()
			root.AddCommand(cli.NewMailCmd(opts))
			var buf bytes.Buffer
			root.SetOut(&buf)
			root.SetErr(&buf)
			root.SetIn(strings.NewReader(""))
			root.SetArgs(append(args,
				"--login", "w0000000", "--auth-data", "x", "--auth-type", "plain"))
			err := root.Execute()
			if err == nil {
				t.Fatalf("Execute %v: want refusal error, got nil", args)
			}
			if !errors.Is(err, cli.ErrConfirmationRequired) {
				t.Errorf("err = %v, want ErrConfirmationRequired", err)
			}
		})
	}
}

func TestMailFiltersHelpListsSubcommands(t *testing.T) {
	t.Parallel()
	root, opts := cli.NewRootCmd()
	root.AddCommand(cli.NewMailCmd(opts))

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"mail", "filters", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"list", "add", "delete"} {
		if !strings.Contains(out, want) {
			t.Errorf("--help output missing %q\n%s", want, out)
		}
	}
}

// add_mailstandardfilter without --filter, or with a malformed --filter
// (empty item, embedded ';'), is rejected at build time before any
// gating, credential resolution OR audit/dry-run record is emitted —
// the KAS API would otherwise fault missing_parameter or accept a
// silently-mangled chain. The dry-run + --yes path is asserted
// explicitly so the validation runs even when the user is wired up for
// non-interactive automation.
func TestMailFiltersAddRejectsBadFilter(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		args []string
	}{
		{
			"no --filter",
			[]string{"mail", "filters", "add", "m0000001"},
		},
		{
			"empty --filter item",
			[]string{"mail", "filters", "add", "m0000001", "--filter", ""},
		},
		{
			"--filter contains ';'",
			[]string{"mail", "filters", "add", "m0000001", "--filter", "pdw;virus_mark"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			root, opts := cli.NewRootCmd()
			root.AddCommand(cli.NewMailCmd(opts))
			var buf bytes.Buffer
			root.SetOut(&buf)
			root.SetErr(&buf)
			// --dry-run + --yes would otherwise skip the prompt and emit
			// an audit record — assert the validator still runs first.
			root.SetArgs(append(append([]string{}, c.args...),
				"--dry-run", "--yes",
				"--login", "w0000000", "--auth-data", "x", "--auth-type", "plain"))
			err := root.Execute()
			if err == nil {
				t.Fatalf("Execute %v: want error, got nil", c.args)
			}
			if cli.CodeFor(err) != cli.ExitUserError {
				t.Errorf("exit code = %d, want ExitUserError", cli.CodeFor(err))
			}
		})
	}
}

// Both destructive subcommands (add — replaces the chain wholesale —
// and delete) must refuse on a non-interactive stdin without --yes
// rather than dispatch unconfirmed.
func TestMailFiltersDestructiveRefuseNonTTY(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"mail", "filters", "add", "m0000001", "--filter", "pdw"},
		{"mail", "filters", "delete", "m0000001"},
	} {
		t.Run(args[2], func(t *testing.T) {
			t.Parallel()
			root, opts := cli.NewRootCmd()
			root.AddCommand(cli.NewMailCmd(opts))
			var buf bytes.Buffer
			root.SetOut(&buf)
			root.SetErr(&buf)
			root.SetIn(strings.NewReader(""))
			root.SetArgs(append(args,
				"--login", "w0000000", "--auth-data", "x", "--auth-type", "plain"))
			err := root.Execute()
			if err == nil {
				t.Fatalf("Execute %v: want refusal error, got nil", args)
			}
			if !errors.Is(err, cli.ErrConfirmationRequired) {
				t.Errorf("err = %v, want ErrConfirmationRequired", err)
			}
		})
	}
}

// The mail filters add chain assembly is the most-likely drift surface
// (repeatable --filter joined with ';'). --dry-run renders the exact
// KAS params it would dispatch as JSON so this asserts the assembly end
// to end without a network call.
func TestMailFiltersAddDryRunChainAssembly(t *testing.T) {
	t.Parallel()
	root, opts := cli.NewRootCmd()
	root.AddCommand(cli.NewMailCmd(opts))
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	root.SetArgs([]string{
		"mail", "filters", "add", "m0000001",
		"--filter", "pdw",
		"--filter", "virus_mark",
		"--filter", "spamc_move:move=Spam",
		"--dry-run", "-o", "json",
		"--login", "w0", "--auth-data", "x", "--auth-type", "plain",
	})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var got struct {
		Action string            `json:"action"`
		Params map[string]string `json:"params"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal dry-run output: %v\n%s", err, out.String())
	}
	if got.Action != "add_mailstandardfilter" {
		t.Errorf("action = %q, want add_mailstandardfilter", got.Action)
	}
	if got.Params["mail_login"] != "m0000001" {
		t.Errorf("params[mail_login] = %q, want m0000001", got.Params["mail_login"])
	}
	if got.Params["filter"] != "pdw;virus_mark;spamc_move:move=Spam" {
		t.Errorf("params[filter] = %q, want pdw;virus_mark;spamc_move:move=Spam", got.Params["filter"])
	}
}

func TestMailListsHelpListsSubcommands(t *testing.T) {
	t.Parallel()
	root, opts := cli.NewRootCmd()
	root.AddCommand(cli.NewMailCmd(opts))

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"mail", "lists", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"list", "get", "add", "update", "delete"} {
		if !strings.Contains(out, want) {
			t.Errorf("--help output missing %q\n%s", want, out)
		}
	}
}

func TestMailListsAddRejectsBadInput(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		args []string
	}{
		{"missing --domain", []string{"mail", "lists", "add", "announce", "--password", "pw"}},
		{"missing --password", []string{"mail", "lists", "add", "announce", "--domain", "example.com"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			root, opts := cli.NewRootCmd()
			root.AddCommand(cli.NewMailCmd(opts))
			var buf bytes.Buffer
			root.SetOut(&buf)
			root.SetErr(&buf)
			root.SetArgs(c.args)
			err := root.Execute()
			if err == nil {
				t.Fatalf("Execute %v: want error, got nil", c.args)
			}
			if cli.CodeFor(err) != cli.ExitUserError {
				t.Errorf("exit code = %d, want ExitUserError", cli.CodeFor(err))
			}
		})
	}
}

// The destructive lists subcommands (update/delete) must refuse on a
// non-interactive stdin without --yes rather than dispatch unconfirmed.
func TestMailListsDestructiveRefuseNonTTY(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"mail", "lists", "delete", "announce-example-com"},
		{"mail", "lists", "update", "announce-example-com", "--active", "Y"},
	} {
		t.Run(args[2], func(t *testing.T) {
			t.Parallel()
			root, opts := cli.NewRootCmd()
			root.AddCommand(cli.NewMailCmd(opts))
			var buf bytes.Buffer
			root.SetOut(&buf)
			root.SetErr(&buf)
			root.SetIn(strings.NewReader(""))
			root.SetArgs(append(args,
				"--login", "w0000000", "--auth-data", "x", "--auth-type", "plain"))
			err := root.Execute()
			if err == nil {
				t.Fatalf("Execute %v: want refusal error, got nil", args)
			}
			if !errors.Is(err, cli.ErrConfirmationRequired) {
				t.Errorf("err = %v, want ErrConfirmationRequired", err)
			}
		})
	}
}

// The mail lists update field assembly is the most intricate part of
// the slice: only flags the user explicitly set are sent (keyed on
// cobra Changed), --active maps to is_active Y/N, and --subscriber /
// --restrict-post repeats join with a newline. --dry-run renders the
// exact KAS params it would dispatch as JSON, so this asserts the
// assembly end to end without a network call. Multi-line values are
// elided by RedactParams before they reach the preview, so the
// newline-join is pinned via the elided byte count (two 6-byte
// addresses + one separator byte = 13).
func TestMailListsUpdateDryRunFieldAssembly(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		args   []string
		want   map[string]string
		absent []string
	}{
		{
			"active Y",
			[]string{"mail", "lists", "update", "L", "--active", "Y"},
			map[string]string{"mailinglist_name": "L", "is_active": "Y"},
			[]string{"subscriber", "restrict_post", "config"},
		},
		{
			"active N",
			[]string{"mail", "lists", "update", "L", "--active", "N"},
			map[string]string{"mailinglist_name": "L", "is_active": "N"},
			nil,
		},
		{
			"subscriber repeats join with newline",
			[]string{"mail", "lists", "update", "L", "--subscriber", "a@x.de", "--subscriber", "b@x.de"},
			map[string]string{"mailinglist_name": "L", "subscriber": "<elided 13 bytes>"},
			[]string{"is_active"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			root, opts := cli.NewRootCmd()
			root.AddCommand(cli.NewMailCmd(opts))
			var out, errb bytes.Buffer
			root.SetOut(&out)
			root.SetErr(&errb)
			args := append(append([]string{}, c.args...),
				"--dry-run", "-o", "json",
				"--login", "w0", "--auth-data", "x", "--auth-type", "plain")
			root.SetArgs(args)
			if err := root.Execute(); err != nil {
				t.Fatalf("Execute %v: %v", args, err)
			}
			var got struct {
				Action string            `json:"action"`
				Params map[string]string `json:"params"`
			}
			if err := json.Unmarshal(out.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal preview: %v\nstdout:%s", err, out.String())
			}
			if got.Action != "update_mailinglist" {
				t.Errorf("action = %q, want update_mailinglist", got.Action)
			}
			for k, v := range c.want {
				if got.Params[k] != v {
					t.Errorf("params[%q] = %q, want %q (full: %v)", k, got.Params[k], v, got.Params)
				}
			}
			for _, k := range c.absent {
				if _, ok := got.Params[k]; ok {
					t.Errorf("params[%q] present, want absent (full: %v)", k, got.Params)
				}
			}
		})
	}
}
