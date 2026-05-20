package cli_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/cli"
)

func TestDatabasesCmdHelpListsSubcommands(t *testing.T) {
	t.Parallel()
	root, opts := cli.NewRootCmd()
	root.AddCommand(cli.NewDatabasesCmd(opts))

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"databases", "--help"})
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

func TestDatabasesAddRejectsBadInput(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		args []string
	}{
		{"missing --password", []string{"databases", "add", "--comment", "c", "--allowed-hosts", "localhost"}},
		{"missing --comment", []string{"databases", "add", "--password", "s3cret", "--allowed-hosts", "localhost"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			root, opts := cli.NewRootCmd()
			root.AddCommand(cli.NewDatabasesCmd(opts))
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

// add must accept omitted --allowed-hosts: an empty value is the KAS
// API's documented "any host may connect" wildcard, not a missing
// parameter. The dry-run preview must therefore reach action /
// params assembly (no validation rejection) and the
// database_allowed_hosts key must be present in the params with the
// empty-string value.
func TestDatabasesAddOptionalAllowedHosts(t *testing.T) {
	t.Parallel()
	root, opts := cli.NewRootCmd()
	root.AddCommand(cli.NewDatabasesCmd(opts))
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	root.SetArgs([]string{
		"databases", "add",
		"--password", "s3cret",
		"--comment", "Test DB",
		"--dry-run", "-o", "json",
		"--login", "w0", "--auth-data", "x", "--auth-type", "plain",
	})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v\nstderr: %s", err, errb.String())
	}
	var got struct {
		Action string            `json:"action"`
		Params map[string]string `json:"params"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal preview: %v\nstdout: %s", err, out.String())
	}
	if got.Action != "add_database" {
		t.Errorf("action = %q, want add_database", got.Action)
	}
	v, ok := got.Params["database_allowed_hosts"]
	if !ok {
		t.Errorf("params missing database_allowed_hosts (empty wildcard must still be sent on the wire): %v", got.Params)
	}
	if v != "" {
		t.Errorf("params[database_allowed_hosts] = %q, want \"\" (wildcard)", v)
	}
}

// The destructive database subcommands (update/delete) must refuse on
// a non-interactive stdin without --yes rather than dispatch
// unconfirmed.
func TestDatabasesDestructiveRefuseNonTTY(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"databases", "delete", "d0123460"},
		{"databases", "update", "d0123460", "--comment", "x"},
	} {
		t.Run(args[1], func(t *testing.T) {
			t.Parallel()
			root, opts := cli.NewRootCmd()
			root.AddCommand(cli.NewDatabasesCmd(opts))
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

// update_database must send only the explicitly-changed flags (keyed
// on cobra Changed), so an unset field never leaks into the request and
// an empty value passed explicitly is a deliberate set. The password
// flag must map to database_new_password (update's key), never the
// add-only database_password.
func TestDatabasesUpdateDryRunFieldAssembly(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		args   []string
		want   map[string]string
		absent []string
	}{
		{
			"comment only",
			[]string{"databases", "update", "d0123460", "--comment", "Renamed"},
			map[string]string{"database_login": "d0123460", "database_comment": "Renamed"},
			[]string{"database_new_password", "database_allowed_hosts"},
		},
		{
			// --password must map to update's database_new_password
			// key, never the add-only database_password. The value
			// itself is redacted in the dry-run/audit preview
			// (RedactParams masks *password*), so the mapping is
			// proven by the key's presence + absence of
			// database_password rather than the cleartext.
			"new password maps to database_new_password (redacted)",
			[]string{"databases", "update", "d0123460", "--password", "n3wpass"},
			map[string]string{"database_login": "d0123460", "database_new_password": "<redacted>"},
			[]string{"database_password", "database_comment"},
		},
		{
			"allowed-hosts only",
			[]string{"databases", "update", "d0123460", "--allowed-hosts", "localhost, 192.168.100.10/24"},
			map[string]string{"database_login": "d0123460", "database_allowed_hosts": "localhost, 192.168.100.10/24"},
			[]string{"database_comment", "database_new_password"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			root, opts := cli.NewRootCmd()
			root.AddCommand(cli.NewDatabasesCmd(opts))
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
			if got.Action != "update_database" {
				t.Errorf("action = %q, want update_database", got.Action)
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

// add and update bind disjoint flag sets. Each set is currently
// flag-name-identical (--password / --comment / --allowed-hosts), but
// the bind is per-subcommand so the help text reflects the action
// semantics (initial vs replacement, required vs optional) and a
// future refactor that re-merges them silently can't sneak past the
// help-text-truthfulness contract. Add a sentinel update-only flag
// here when the action surfaces diverge (ddnsuser-style).
func TestDatabasesFlagSetsAreDisjoint(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		args []string
	}{
		// Sentinel: passing an unknown flag (--target-ipv4) on add or
		// update must fail at cobra parse time rather than be silently
		// ignored. Identical add/update surfaces today still rely on
		// per-subcommand bind, so re-merging would re-introduce the
		// help-text drift the ddnsuser slice already fixed.
		{"unknown --target-ipv4 rejected on add", []string{
			"databases", "add",
			"--password", "s3cret", "--comment", "c", "--allowed-hosts", "localhost",
			"--target-ipv4", "127.0.0.1",
		}},
		{"unknown --target-ipv4 rejected on update", []string{
			"databases", "update", "d0123460",
			"--target-ipv4", "127.0.0.1",
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			root, opts := cli.NewRootCmd()
			root.AddCommand(cli.NewDatabasesCmd(opts))
			var buf bytes.Buffer
			root.SetOut(&buf)
			root.SetErr(&buf)
			root.SetArgs(c.args)
			err := root.Execute()
			if err == nil {
				t.Fatalf("Execute %v: want unknown-flag error, got nil", c.args)
			}
			if !strings.Contains(err.Error(), "unknown flag") {
				t.Errorf("err = %q, want it to contain 'unknown flag'", err)
			}
		})
	}
}

// The delete_database ConfirmAction uses the louder "permanently
// delete" verb instead of the bare "delete" every other slice uses.
// Pin both the verb and the rendered Summary so a future refactor of
// either the slice or the shared ConfirmAction template cannot
// silently regress the loudness contract — the source-code comment
// alone is not enforceable.
func TestDatabasesDeleteConfirmIsLouder(t *testing.T) {
	t.Parallel()
	a := cli.DatabaseDeleteConfirm("d0123460")
	if a.Verb != "permanently delete" {
		t.Errorf("Verb = %q, want %q", a.Verb, "permanently delete")
	}
	if a.Resource != "database" {
		t.Errorf("Resource = %q, want %q", a.Resource, "database")
	}
	if a.ID != "d0123460" {
		t.Errorf("ID = %q, want d0123460", a.ID)
	}
	want := `About to permanently delete database "d0123460". This cannot be undone.`
	if got := a.Summary(); got != want {
		t.Errorf("Summary() = %q, want %q", got, want)
	}
}

// On --dry-run the runWriteE seam must still emit a #131 audit record
// (outcome=dry-run, action=delete_database, target=<login>,
// database_login=<login>) on stderr, even though no SOAP call is
// dispatched. This pins the database delete subcommand's wiring into
// the audit emission path — without it, a future refactor that breaks
// the runWriteE → WriteAudit glue would only be caught at the
// surrounding-package level.
func TestDatabasesDeleteDryRunEmitsAuditLine(t *testing.T) {
	t.Parallel()
	root, opts := cli.NewRootCmd()
	root.AddCommand(cli.NewDatabasesCmd(opts))
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	root.SetArgs([]string{
		"databases", "delete", "d0123460",
		"--dry-run",
		"--login", "w0000000", "--auth-data", "x", "--auth-type", "plain",
	})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	line := errb.String()
	for _, want := range []string{
		"action=delete_database",
		"target=d0123460",
		"outcome=dry-run",
		"database_login=d0123460",
		"login=w0000000",
	} {
		if !strings.Contains(line, want) {
			t.Errorf("audit line missing %q\nline: %s", want, line)
		}
	}
}

// delete_database's dry-run preview must address the database by its
// login (target=<login>) and use the delete_database action verbatim,
// so the audit trail can later be reconciled to the resource that was
// dropped.
func TestDatabasesDeleteDryRunTargetsLogin(t *testing.T) {
	t.Parallel()
	root, opts := cli.NewRootCmd()
	root.AddCommand(cli.NewDatabasesCmd(opts))
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	root.SetArgs([]string{
		"databases", "delete", "d0123460",
		"--dry-run", "-o", "json",
		"--login", "w0", "--auth-data", "x", "--auth-type", "plain",
	})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var got struct {
		Action string            `json:"action"`
		Target string            `json:"target"`
		Params map[string]string `json:"params"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal preview: %v\nstdout: %s", err, out.String())
	}
	if got.Action != "delete_database" {
		t.Errorf("action = %q, want delete_database", got.Action)
	}
	if got.Target != "d0123460" {
		t.Errorf("target = %q, want d0123460", got.Target)
	}
	if got.Params["database_login"] != "d0123460" {
		t.Errorf("params[database_login] = %q, want d0123460 (full: %v)", got.Params["database_login"], got.Params)
	}
}
