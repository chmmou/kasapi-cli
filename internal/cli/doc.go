// Package cli holds the kasapi-cli root command, global flags, and the
// shared output renderers (json / yaml / table).
//
// The package is the outermost adapter on top of internal/api, internal/auth
// and internal/config: subcommands (added with #7+) translate flags and args
// into use-case calls and feed the typed result to a Renderer chosen by the
// global --output flag.
//
// See issue #12.
package cli
