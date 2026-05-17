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
	for _, want := range []string{"list", "get"} {
		if !strings.Contains(out, want) {
			t.Errorf("--help output missing %q\n%s", want, out)
		}
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
	if !strings.Contains(out, "list") {
		t.Errorf("--help output missing %q\n%s", "list", out)
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
		{"mail", "lists", "update", "announce-example-com", "--active"},
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
// assembly end to end without a network call.
func TestMailListsUpdateDryRunFieldAssembly(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		args   []string
		want   map[string]string
		absent []string
	}{
		{
			"active true",
			[]string{"mail", "lists", "update", "L", "--active"},
			map[string]string{"mailinglist_name": "L", "is_active": "Y"},
			[]string{"subscriber", "restrict_post", "config"},
		},
		{
			"active false",
			[]string{"mail", "lists", "update", "L", "--active=false"},
			map[string]string{"mailinglist_name": "L", "is_active": "N"},
			nil,
		},
		{
			"subscriber repeats join with newline",
			[]string{"mail", "lists", "update", "L", "--subscriber", "a@x.de", "--subscriber", "b@x.de"},
			map[string]string{"mailinglist_name": "L", "subscriber": "a@x.de\nb@x.de"},
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
