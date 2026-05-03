// Command kasapi-cli is the All-Inkl KAS API command-line client.
//
// This is the wiring layer per docs/go/ARCHITECTURE.md: it composes the
// inner layers (domain/use cases) with the outer adapters (SOAP transport,
// HTTP client, CLI) and exposes them through subcommands.
package main

import (
	"fmt"
	"os"

	"github.com/chmmou/kasapi-cli/internal/cli"
)

func main() {
	root, opts := cli.NewRootCmd()
	root.AddCommand(
		cli.NewAccountCmd(opts),
		cli.NewServerCmd(opts),
		cli.NewConfigCmd(opts),
	)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "kasapi-cli:", err)
		os.Exit(cli.CodeFor(err))
	}
}
