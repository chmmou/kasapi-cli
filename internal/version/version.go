// Package version exposes build-time version information for kasapi-cli.
//
// Version, Commit and Date are populated via -ldflags at build time:
//
//	go build -ldflags "-X github.com/chmmou/kasapi-cli/internal/version.Version=v0.1.0 \
//	                   -X github.com/chmmou/kasapi-cli/internal/version.Commit=$(git rev-parse --short HEAD) \
//	                   -X github.com/chmmou/kasapi-cli/internal/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
//	  ./cmd/kasapi-cli
package version

import "fmt"

// Version is the semver tag the binary was built from.
var Version = "dev"

// Commit is the short git revision the binary was built from.
var Commit = "none"

// Date is the build timestamp in RFC3339 form.
var Date = "unknown"

// String returns a human-readable single-line version banner.
func String() string {
	return fmt.Sprintf("kasapi-cli %s (commit %s, built %s)", Version, Commit, Date)
}
