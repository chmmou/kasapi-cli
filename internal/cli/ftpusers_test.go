package cli_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/cli"
)

func TestFTPUsersCmdHelpListsSubcommands(t *testing.T) {
	t.Parallel()
	root, opts := cli.NewRootCmd()
	root.AddCommand(cli.NewFTPUsersCmd(opts))

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"ftpusers", "--help"})
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

func TestFTPUsersAddRejectsBadInput(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		args []string
	}{
		{"missing --password", []string{"ftpusers", "add", "--comment", "c"}},
		{"missing --comment", []string{"ftpusers", "add", "--password", "s3cret"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			root, opts := cli.NewRootCmd()
			root.AddCommand(cli.NewFTPUsersCmd(opts))
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

// The destructive ftpuser subcommands (update/delete) must refuse on a
// non-interactive stdin without --yes rather than dispatch unconfirmed.
func TestFTPUsersDestructiveRefuseNonTTY(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"ftpusers", "delete", "f0000004"},
		{"ftpusers", "update", "f0000004", "--comment", "x"},
	} {
		t.Run(args[1], func(t *testing.T) {
			t.Parallel()
			root, opts := cli.NewRootCmd()
			root.AddCommand(cli.NewFTPUsersCmd(opts))
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

// update_ftpuser must send only the explicitly-changed flags (keyed on
// cobra Changed), so an unset field never leaks into the request and an
// empty value passed explicitly is a deliberate set. The password flag
// must map to ftp_new_password (update's key), never ftp_password.
func TestFTPUsersUpdateDryRunFieldAssembly(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		args   []string
		want   map[string]string
		absent []string
	}{
		{
			"comment only",
			[]string{"ftpusers", "update", "f0000004", "--comment", "Renamed"},
			map[string]string{"ftp_login": "f0000004", "ftp_comment": "Renamed"},
			[]string{"ftp_new_password", "ftp_path", "ftp_permission_read"},
		},
		{
			// --password must map to update's ftp_new_password key, never
			// the add-only ftp_password. The value itself is redacted in
			// the dry-run/audit preview (RedactParams masks *password*),
			// so the mapping is proven by the key's presence + absence of
			// ftp_password rather than the cleartext.
			"new password maps to ftp_new_password (redacted)",
			[]string{"ftpusers", "update", "f0000004", "--password", "n3wpass"},
			map[string]string{"ftp_login": "f0000004", "ftp_new_password": "<redacted>"},
			[]string{"ftp_password", "ftp_comment"},
		},
		{
			"deny write",
			[]string{"ftpusers", "update", "f0000004", "--permission-write=false"},
			map[string]string{"ftp_login": "f0000004", "ftp_permission_write": "N"},
			[]string{"ftp_permission_read", "ftp_comment"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			root, opts := cli.NewRootCmd()
			root.AddCommand(cli.NewFTPUsersCmd(opts))
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
			if got.Action != "update_ftpuser" {
				t.Errorf("action = %q, want update_ftpuser", got.Action)
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
