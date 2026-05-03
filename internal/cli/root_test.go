package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/cli"
)

func TestRootHelpListsAllPersistentFlags(t *testing.T) {
	t.Parallel()
	root, _ := cli.NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute --help: %v", err)
	}
	got := out.String()
	for _, flag := range []string{
		"--config", "--profile", "--login", "--auth-data", "--auth-type",
		"--output", "--no-color", "--verbose", "--yes",
	} {
		if !strings.Contains(got, flag) {
			t.Errorf("help is missing flag %s; got:\n%s", flag, got)
		}
	}
}

func TestRootBindsFlagsToOptions(t *testing.T) {
	t.Parallel()
	root, opts := cli.NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{
		"--config", "/tmp/cfg.toml",
		"--profile", "prod",
		"--login", "w012345",
		"--auth-data", "secret",
		"--auth-type", "session",
		"--output", "json",
		"--no-color",
		"--verbose",
		"--yes",
	})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if opts.ConfigPath != "/tmp/cfg.toml" {
		t.Errorf("ConfigPath = %q", opts.ConfigPath)
	}
	if opts.Profile != "prod" {
		t.Errorf("Profile = %q", opts.Profile)
	}
	if opts.Login != "w012345" || opts.AuthData != "secret" || opts.AuthType != "session" {
		t.Errorf("credential overrides not bound: %+v", opts)
	}
	if opts.Output != cli.FormatJSON {
		t.Errorf("Output = %q, want json", opts.Output)
	}
	if !opts.NoColor || !opts.Verbose || !opts.Yes {
		t.Errorf("bool flags not bound: NoColor=%v Verbose=%v Yes=%v", opts.NoColor, opts.Verbose, opts.Yes)
	}
}

func TestRootRejectsInvalidOutputFormat(t *testing.T) {
	t.Parallel()
	root, _ := cli.NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--output", "xml"})
	err := root.Execute()
	if err == nil {
		t.Fatal("Execute --output xml: want error, got nil")
	}
	if cli.CodeFor(err) != cli.ExitUserError {
		t.Errorf("invalid --output should map to ExitUserError, got %d", cli.CodeFor(err))
	}
	if !strings.Contains(err.Error(), "invalid output format") {
		t.Errorf("error message: %q", err.Error())
	}
}

func TestRootRejectsUnknownFlag(t *testing.T) {
	t.Parallel()
	root, _ := cli.NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--frobnicate"})
	err := root.Execute()
	if err == nil {
		t.Fatal("Execute --frobnicate: want error, got nil")
	}
	if cli.CodeFor(err) != cli.ExitUserError {
		t.Errorf("unknown flag should map to ExitUserError, got %d", cli.CodeFor(err))
	}
}

func TestRootDefaultOutputIsTable(t *testing.T) {
	t.Parallel()
	root, opts := cli.NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(nil)
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if opts.Output != cli.FormatTable {
		t.Errorf("default Output = %q, want %q", opts.Output, cli.FormatTable)
	}
}
