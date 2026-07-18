package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/cli"
)

func TestRootHelpListsVisiblePersistentFlags(t *testing.T) {
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
		"--output", "--verbose", "--yes", "--audit-log", "--dry-run",
	} {
		if !strings.Contains(got, flag) {
			t.Errorf("help is missing flag %s; got:\n%s", flag, got)
		}
	}
	// --yes is now wired (issue #109, destructive-write gate) so it is
	// advertised again. --no-color stays reserved/unwired and must not
	// be advertised in --help yet. TestRootBindsFlagsToOptions still
	// proves --no-color parses.
	if strings.Contains(got, "--no-color") {
		t.Errorf("help advertises unwired flag --no-color; want it hidden until implemented; got:\n%s", got)
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

// TestRootUnknownCommandExitsUserError pins the exit-code contract for
// a mistyped subcommand: bad user input must map to exit 1, not fall
// through CodeFor to the API-error exit 2.
func TestRootUnknownCommandExitsUserError(t *testing.T) {
	t.Parallel()
	root, _ := cli.NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"frobnicate"})
	err := root.Execute()
	if err == nil {
		t.Fatal("Execute frobnicate: want error, got nil")
	}
	if cli.CodeFor(err) != cli.ExitUserError {
		t.Errorf("unknown command should map to ExitUserError, got %d", cli.CodeFor(err))
	}
	if !strings.Contains(err.Error(), `unknown command "frobnicate"`) {
		t.Errorf("error message: %q", err.Error())
	}
}

// TestGroupUnknownSubcommandExitsUserError pins the exit-code contract
// for a typo'd subcommand under a group command: cobra alone treats it
// as a bare help call and exits 0, so Finalize must reject it as a
// user error (exit 1).
func TestGroupUnknownSubcommandExitsUserError(t *testing.T) {
	t.Parallel()
	root, opts := cli.NewRootCmd()
	root.AddCommand(cli.NewMailCmd(opts))
	cli.Finalize(root)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"mail", "frobnicate"})
	err := root.Execute()
	if err == nil {
		t.Fatal("Execute mail frobnicate: want error, got nil")
	}
	if cli.CodeFor(err) != cli.ExitUserError {
		t.Errorf("unknown subcommand should map to ExitUserError, got %d", cli.CodeFor(err))
	}
	if !strings.Contains(err.Error(), `unknown command "frobnicate"`) {
		t.Errorf("error message: %q", err.Error())
	}
}

// A bare group invocation stays a help call with exit 0 after Finalize.
func TestGroupBareInvocationPrintsHelp(t *testing.T) {
	t.Parallel()
	root, opts := cli.NewRootCmd()
	root.AddCommand(cli.NewMailCmd(opts))
	cli.Finalize(root)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"mail"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute mail: %v", err)
	}
	if !strings.Contains(out.String(), "accounts") {
		t.Errorf("group help output missing subcommands:\n%s", out.String())
	}
}

// TestCompletionArgErrorsExitUserError pins that the lazily-registered
// completion command is covered too: Finalize registers it before the
// walkers run, so its args-validation failures exit 1, not 2.
func TestCompletionArgErrorsExitUserError(t *testing.T) {
	t.Parallel()
	root, opts := cli.NewRootCmd()
	root.AddCommand(cli.NewDomainsCmd(opts))
	cli.Finalize(root)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"completion", "bash", "extra"})
	err := root.Execute()
	if err == nil {
		t.Fatal("Execute completion bash extra: want error, got nil")
	}
	if cli.CodeFor(err) != cli.ExitUserError {
		t.Errorf("completion args error should map to ExitUserError, got %d", cli.CodeFor(err))
	}
}

// TestHelpUnknownTopicExitsUserError pins that `help <nonsense>` exits
// 1 like every other unknown command: cobra's stock help command
// returns nil for an unresolvable topic, which would read as success
// to scripts.
func TestHelpUnknownTopicExitsUserError(t *testing.T) {
	t.Parallel()
	root, opts := cli.NewRootCmd()
	root.AddCommand(cli.NewMailCmd(opts))
	cli.Finalize(root)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"help", "frobnicate"})
	err := root.Execute()
	if err == nil {
		t.Fatal("Execute help frobnicate: want error, got nil")
	}
	if cli.CodeFor(err) != cli.ExitUserError {
		t.Errorf("unknown help topic should map to ExitUserError, got %d", cli.CodeFor(err))
	}
	if !strings.Contains(err.Error(), "unknown help topic") {
		t.Errorf("error message: %q", err.Error())
	}
}

// `help <group> <subcommand>` and bare `help` keep printing help with
// exit 0 after the unknown-topic hardening.
func TestHelpKnownTopicsPrintHelp(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"help"},
		{"help", "mail"},
		{"help", "mail", "accounts"},
	} {
		root, opts := cli.NewRootCmd()
		root.AddCommand(cli.NewMailCmd(opts))
		cli.Finalize(root)
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)
		root.SetArgs(args)
		if err := root.Execute(); err != nil {
			t.Fatalf("Execute %v: %v", args, err)
		}
		if out.Len() == 0 {
			t.Errorf("Execute %v printed no help output", args)
		}
	}
}

// TestMarkArgErrorsAsUserErrors pins the exit-code contract for a
// positional-args validation failure (e.g. a missing required argument
// on an ExactArgs(1) subcommand) after the cmd/kasapi-cli wiring has
// applied the tree-wide wrapper.
func TestMarkArgErrorsAsUserErrors(t *testing.T) {
	t.Parallel()
	root, opts := cli.NewRootCmd()
	root.AddCommand(cli.NewDomainsCmd(opts))
	cli.MarkArgErrorsAsUserErrors(root)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"domains", "get"})
	err := root.Execute()
	if err == nil {
		t.Fatal("Execute domains get (missing arg): want error, got nil")
	}
	if cli.CodeFor(err) != cli.ExitUserError {
		t.Errorf("missing positional arg should map to ExitUserError, got %d", cli.CodeFor(err))
	}
}

// TestConfigInitDoesNotShadowRootProfile guards against the local
// --profile flag on `config init` reappearing and silently overriding
// the persistent root --profile. The local flag is now --name; the
// root --profile must still bind to opts.Profile when the subcommand
// runs.
func TestConfigInitDoesNotShadowRootProfile(t *testing.T) {
	t.Parallel()
	root, opts := cli.NewRootCmd()
	root.AddCommand(cli.NewConfigCmd(opts))
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	// `config init` requires a TTY → it will exit with a user error before
	// any prompt. We only need to assert that flag parsing succeeded and
	// the root --profile reached opts.
	root.SetArgs([]string{"--profile", "prod", "config", "init", "--name", "staging"})
	_ = root.Execute()
	if opts.Profile != "prod" {
		t.Errorf("root --profile = %q, want prod (was the local --name flag shadowing it?)", opts.Profile)
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
