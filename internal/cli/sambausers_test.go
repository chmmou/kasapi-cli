package cli_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/cli"
)

func TestSambaUsersCmdHelpListsSubcommands(t *testing.T) {
	t.Parallel()
	root, opts := cli.NewRootCmd()
	root.AddCommand(cli.NewSambaUsersCmd(opts))

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"sambausers", "--help"})
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

func TestSambaUsersAddRejectsBadInput(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		args []string
	}{
		{"missing --password", []string{"sambausers", "add", "--comment", "c", "--path", "/p/"}},
		{"missing --comment", []string{"sambausers", "add", "--password", "s3cret", "--path", "/p/"}},
		{"missing --path", []string{"sambausers", "add", "--password", "s3cret", "--comment", "c"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			root, opts := cli.NewRootCmd()
			root.AddCommand(cli.NewSambaUsersCmd(opts))
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

// The destructive sambauser subcommands (update/delete) must refuse on
// a non-interactive stdin without --yes rather than dispatch
// unconfirmed.
func TestSambaUsersDestructiveRefuseNonTTY(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"sambausers", "delete", "s0000000"},
		{"sambausers", "update", "s0000000", "--comment", "x"},
	} {
		t.Run(args[1], func(t *testing.T) {
			t.Parallel()
			root, opts := cli.NewRootCmd()
			root.AddCommand(cli.NewSambaUsersCmd(opts))
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

// update_sambauser must send only the explicitly-changed flags (keyed
// on cobra Changed), so an unset field never leaks into the request and
// an empty value passed explicitly is a deliberate set. The password
// flag must map to samba_new_password (update's key), never the
// add-only samba_password.
func TestSambaUsersUpdateDryRunFieldAssembly(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		args   []string
		want   map[string]string
		absent []string
	}{
		{
			"comment only",
			[]string{"sambausers", "update", "s0000000", "--comment", "Renamed"},
			map[string]string{"samba_login": "s0000000", "samba_comment": "Renamed"},
			[]string{"samba_new_password", "samba_path"},
		},
		{
			// --password must map to update's samba_new_password key,
			// never the add-only samba_password. The value itself is
			// redacted in the dry-run/audit preview (RedactParams masks
			// *password*), so the mapping is proven by the key's
			// presence + absence of samba_password rather than the
			// cleartext.
			"new password maps to samba_new_password (redacted)",
			[]string{"sambausers", "update", "s0000000", "--password", "n3wpass"},
			map[string]string{"samba_login": "s0000000", "samba_new_password": "<redacted>"},
			[]string{"samba_password", "samba_comment"},
		},
		{
			"path only",
			[]string{"sambausers", "update", "s0000000", "--path", "/example.com/share-renamed/"},
			map[string]string{"samba_login": "s0000000", "samba_path": "/example.com/share-renamed/"},
			[]string{"samba_comment", "samba_new_password"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			root, opts := cli.NewRootCmd()
			root.AddCommand(cli.NewSambaUsersCmd(opts))
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
			if got.Action != "update_sambauser" {
				t.Errorf("action = %q, want update_sambauser", got.Action)
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
