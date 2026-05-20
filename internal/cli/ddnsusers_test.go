package cli_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/cli"
)

func TestDDNSUsersCmdHelpListsSubcommands(t *testing.T) {
	t.Parallel()
	root, opts := cli.NewRootCmd()
	root.AddCommand(cli.NewDDNSUsersCmd(opts))

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"ddnsusers", "--help"})
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

func TestDDNSUsersAddRejectsBadInput(t *testing.T) {
	t.Parallel()
	complete := []string{
		"--password", "s3cret",
		"--zone", "example.com",
		"--label", "home",
		"--target-ip", "203.0.113.42",
		"--comment", "Home router",
	}
	cases := []struct {
		name string
		drop string
	}{
		{"missing --password", "--password"},
		{"missing --zone", "--zone"},
		{"missing --label", "--label"},
		{"missing --target-ip", "--target-ip"},
		{"missing --comment", "--comment"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			args := []string{"ddnsusers", "add"}
			for i := 0; i < len(complete); i += 2 {
				if complete[i] == c.drop {
					continue
				}
				args = append(args, complete[i], complete[i+1])
			}
			root, opts := cli.NewRootCmd()
			root.AddCommand(cli.NewDDNSUsersCmd(opts))
			var buf bytes.Buffer
			root.SetOut(&buf)
			root.SetErr(&buf)
			root.SetArgs(args)
			err := root.Execute()
			if err == nil {
				t.Fatalf("Execute %v: want error, got nil", args)
			}
			if cli.CodeFor(err) != cli.ExitUserError {
				t.Errorf("exit code = %d, want ExitUserError", cli.CodeFor(err))
			}
		})
	}
}

// The destructive ddnsuser subcommands (update/delete) must refuse on a
// non-interactive stdin without --yes rather than dispatch unconfirmed.
func TestDDNSUsersDestructiveRefuseNonTTY(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"ddnsusers", "delete", "dyn0000001"},
		{"ddnsusers", "update", "dyn0000001", "--comment", "x"},
	} {
		t.Run(args[1], func(t *testing.T) {
			t.Parallel()
			root, opts := cli.NewRootCmd()
			root.AddCommand(cli.NewDDNSUsersCmd(opts))
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

// update_ddnsuser must send only the explicitly-changed flags (keyed on
// cobra Changed), so an unset field never leaks into the request and an
// empty value passed explicitly is a deliberate set. The password flag
// must map to dyndns_password (no _new_password split exists here),
// --target-ipv4/--target-ipv6 map to the undocumented-but-verified
// dual-stack keys.
func TestDDNSUsersUpdateDryRunFieldAssembly(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		args   []string
		want   map[string]string
		absent []string
	}{
		{
			"comment only",
			[]string{"ddnsusers", "update", "dyn0000001", "--comment", "Home router (renamed)"},
			map[string]string{"dyndns_login": "dyn0000001", "dyndns_comment": "Home router (renamed)"},
			[]string{"dyndns_password", "dyndns_target_ipv4", "dyndns_target_ipv6", "dyndns_dual_stack", "dyndns_zone"},
		},
		{
			// --password sends dyndns_password (the same key add uses,
			// no _new_password split here). The value itself is
			// redacted in the dry-run/audit preview (RedactParams masks
			// *password*), so the mapping is proven by the key's
			// presence rather than the cleartext.
			"password maps to dyndns_password (redacted)",
			[]string{"ddnsusers", "update", "dyn0000001", "--password", "n3wpass"},
			map[string]string{"dyndns_login": "dyn0000001", "dyndns_password": "<redacted>"},
			[]string{"dyndns_new_password", "dyndns_comment"},
		},
		{
			"dual-stack ipv4 + ipv6",
			[]string{
				"ddnsusers", "update", "dyn0000001",
				"--target-ipv4", "127.0.0.1",
				"--target-ipv6", "::ffff:7f00:1",
				"--dual-stack",
			},
			map[string]string{
				"dyndns_login":       "dyn0000001",
				"dyndns_target_ipv4": "127.0.0.1",
				"dyndns_target_ipv6": "::ffff:7f00:1",
				"dyndns_dual_stack":  "Y",
			},
			[]string{"dyndns_target_ip", "dyndns_comment"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			root, opts := cli.NewRootCmd()
			root.AddCommand(cli.NewDDNSUsersCmd(opts))
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
			if got.Action != "update_ddnsuser" {
				t.Errorf("action = %q, want update_ddnsuser", got.Action)
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

// add and update bind disjoint flag sets. Passing an add-only flag
// (--zone / --label / --target-ip) to update must fail at cobra parse
// time with "unknown flag", and vice versa for update-only flags
// (--target-ipv4 / --target-ipv6) on add. This pins the boundary so a
// future refactor that merges the bindings cannot silently re-introduce
// the help-text-unity-with-silenced-flags footgun.
func TestDDNSUsersFlagSetsAreDisjoint(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		args []string
	}{
		{"--zone rejected on update", []string{"ddnsusers", "update", "dyn0000001", "--zone", "example.com"}},
		{"--label rejected on update", []string{"ddnsusers", "update", "dyn0000001", "--label", "home"}},
		{"--target-ip rejected on update", []string{"ddnsusers", "update", "dyn0000001", "--target-ip", "127.0.0.1"}},
		{"--target-ipv4 rejected on add", []string{
			"ddnsusers", "add",
			"--password", "s3cret", "--zone", "example.com", "--label", "home",
			"--target-ip", "127.0.0.1", "--comment", "c",
			"--target-ipv4", "127.0.0.1",
		}},
		{"--target-ipv6 rejected on add", []string{
			"ddnsusers", "add",
			"--password", "s3cret", "--zone", "example.com", "--label", "home",
			"--target-ip", "127.0.0.1", "--comment", "c",
			"--target-ipv6", "::1",
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			root, opts := cli.NewRootCmd()
			root.AddCommand(cli.NewDDNSUsersCmd(opts))
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
