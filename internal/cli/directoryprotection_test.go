package cli_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/cli"
)

func TestDirectoryProtectionCmdHelpListsSubcommands(t *testing.T) {
	t.Parallel()
	root, opts := cli.NewRootCmd()
	root.AddCommand(cli.NewDirectoryProtectionCmd(opts))

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"directoryprotection", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"list", "add", "update", "delete"} {
		if !strings.Contains(out, want) {
			t.Errorf("--help output missing %q\n%s", want, out)
		}
	}
}

func TestDirectoryProtectionListHelpAdvertisesPathFlag(t *testing.T) {
	t.Parallel()
	root, opts := cli.NewRootCmd()
	root.AddCommand(cli.NewDirectoryProtectionCmd(opts))

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"directoryprotection", "list", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(buf.String(), "--path") {
		t.Errorf("list --help missing --path flag:\n%s", buf.String())
	}
}

// add requires --password; the (path, user) identity comes from the two
// positional args, so a missing password must fail as a user error
// before any credentials are resolved.
func TestDirectoryProtectionAddRejectsMissingPassword(t *testing.T) {
	t.Parallel()
	root, opts := cli.NewRootCmd()
	root.AddCommand(cli.NewDirectoryProtectionCmd(opts))
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"directoryprotection", "add", "/protected/directory/", "protected_user"})
	err := root.Execute()
	if err == nil {
		t.Fatal("Execute: want error, got nil")
	}
	if cli.CodeFor(err) != cli.ExitUserError {
		t.Errorf("exit code = %d, want ExitUserError", cli.CodeFor(err))
	}
}

// update must send only the explicitly-changed flags (keyed on cobra
// Changed): a password-only update must not leak directory_authname,
// and an authname-only update must not leak directory_password (so an
// omitted password keeps the current one). The password value is
// redacted in the dry-run/audit preview, so the mapping is proven by
// the key's presence + the absence of the other field.
func TestDirectoryProtectionUpdateDryRunFieldAssembly(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		args   []string
		want   map[string]string
		absent []string
	}{
		{
			"password only (redacted)",
			[]string{"directoryprotection", "update", "/protected/directory/", "protected_user", "--password", "n3wpass"},
			map[string]string{
				"directory_path":     "/protected/directory/",
				"directory_user":     "protected_user",
				"directory_password": "<redacted>",
			},
			[]string{"directory_authname"},
		},
		{
			"authname only",
			[]string{"directoryprotection", "update", "/protected/directory/", "protected_user", "--authname", "Realm Only"},
			map[string]string{
				"directory_path":     "/protected/directory/",
				"directory_user":     "protected_user",
				"directory_authname": "Realm Only",
			},
			[]string{"directory_password"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			root, opts := cli.NewRootCmd()
			root.AddCommand(cli.NewDirectoryProtectionCmd(opts))
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
			if got.Action != "update_directoryprotection" {
				t.Errorf("action = %q, want update_directoryprotection", got.Action)
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

// The destructive subcommands (update/delete) must refuse on a
// non-interactive stdin without --yes rather than dispatch unconfirmed.
func TestDirectoryProtectionDestructiveRefuseNonTTY(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"directoryprotection", "delete", "/protected/directory/", "protected_user"},
		{"directoryprotection", "update", "/protected/directory/", "protected_user", "--authname", "x"},
	} {
		t.Run(args[1], func(t *testing.T) {
			t.Parallel()
			root, opts := cli.NewRootCmd()
			root.AddCommand(cli.NewDirectoryProtectionCmd(opts))
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

// On --dry-run the runWriteE seam must still emit a #131 audit record
// (outcome=dry-run, action=delete_directoryprotection, identity fields)
// on stderr even though no SOAP call is dispatched, pinning the delete
// subcommand's wiring into the audit emission path.
func TestDirectoryProtectionDeleteDryRunEmitsAuditLine(t *testing.T) {
	t.Parallel()
	root, opts := cli.NewRootCmd()
	root.AddCommand(cli.NewDirectoryProtectionCmd(opts))
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	root.SetArgs([]string{
		"directoryprotection", "delete", "/protected/directory/", "protected_user",
		"--dry-run",
		"--login", "w0000000", "--auth-data", "x", "--auth-type", "plain",
	})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	line := errb.String()
	for _, want := range []string{
		"action=delete_directoryprotection",
		"outcome=dry-run",
		"directory_path=/protected/directory/",
		"directory_user=protected_user",
		"login=w0000000",
	} {
		if !strings.Contains(line, want) {
			t.Errorf("audit line missing %q\nline: %s", want, line)
		}
	}
}
