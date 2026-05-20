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
		{"missing --allowed-hosts", []string{"databases", "add", "--password", "s3cret", "--comment", "c"}},
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

// delete_database's dry-run preview must address the database by its
// login (target=<login>) and use the delete_database action verbatim,
// so the audit trail can later be reconciled to the resource that was
// dropped. The "louder" prompt verb itself ("permanently delete") is
// pinned by the source-code review anchor (database.go) rather than
// this CLI test because the dry-run preview JSON intentionally omits
// the ConfirmAction shape — only the action / target / params are
// machine-readable.
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
