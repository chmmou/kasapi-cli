package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/cli"
)

func TestDNSCmdHelpListsSubcommands(t *testing.T) {
	t.Parallel()
	root, opts := cli.NewRootCmd()
	root.AddCommand(cli.NewDNSCmd(opts))

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"dns", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "list") {
		t.Errorf("--help output missing %q\n%s", "list", out)
	}
}

// get_dns_settings takes an optional record_id filter, not a
// nameserver. Guard the flag rename so the wrong parameter cannot
// creep back into the read slice.
func TestDNSListExposesRecordIDFlag(t *testing.T) {
	t.Parallel()
	root, opts := cli.NewRootCmd()
	root.AddCommand(cli.NewDNSCmd(opts))

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"dns", "list", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "--record-id") {
		t.Errorf("dns list --help missing --record-id flag\n%s", out)
	}
	if strings.Contains(out, "--nameserver") {
		t.Errorf("dns list --help still exposes the wrong --nameserver flag\n%s", out)
	}
}
