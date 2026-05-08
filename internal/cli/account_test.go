package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/cli"
)

func TestAccountCmdHelpListsSubcommands(t *testing.T) {
	t.Parallel()
	root, opts := cli.NewRootCmd()
	root.AddCommand(cli.NewAccountCmd(opts))

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"accounts", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"list", "get", "settings", "resources"} {
		if !strings.Contains(out, want) {
			t.Errorf("--help output missing %q\n%s", want, out)
		}
	}
}

func TestServerCmdHelpListsInfo(t *testing.T) {
	t.Parallel()
	root, opts := cli.NewRootCmd()
	root.AddCommand(cli.NewServerCmd(opts))

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"server", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(buf.String(), "info") {
		t.Errorf("server --help missing 'info' subcommand:\n%s", buf.String())
	}
}
