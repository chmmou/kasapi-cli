// Package config loads KAS credentials and CLI defaults from a TOML
// file (XDG path), environment variables, and command-line flags, and
// resolves the effective values for a single API call.
//
// Selection precedence (highest first): command-line flag, environment
// variable, named profile from the config file, default profile from
// the config file. Missing required fields are reported by Resolve as
// an error rather than silently filled in.
//
// Credentials redact the auth_data field in their String method so
// secrets do not appear in the default log output or in --help.
//
// File format:
//
//	default_profile = "main"
//
//	[profiles.main]
//	login     = "w0000000"
//	auth_data = "..."
//	auth_type = "session"   # or "plain"
//
// See issue #2.
package config
