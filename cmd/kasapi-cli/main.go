// Command kasapi-cli is the All-Inkl KAS API command-line client.
//
// This is the wiring layer per docs/go/ARCHITECTURE.md: it composes the
// inner layers (domain/use cases) with the outer adapters (SOAP transport,
// HTTP client, CLI) and exposes them through subcommands.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/chmmou/kasapi-cli/internal/version"
)

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version.String())
		return
	}

	fmt.Fprintln(os.Stderr, "kasapi-cli: no subcommand wired up yet (see issues #2-#13)")
	os.Exit(1)
}
