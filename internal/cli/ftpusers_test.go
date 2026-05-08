package cli_test

import (
	"bytes"
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
	for _, want := range []string{"list", "get"} {
		if !strings.Contains(out, want) {
			t.Errorf("--help output missing %q\n%s", want, out)
		}
	}
}
