package cli

// Internal hooks exported for tests in the cli_test package.

// ConfigIO mirrors the unexported configIO struct used by `config init`.
// Tests construct one to drive prompts and password input without a TTY.
type ConfigIO = configIO

// RunConfigInit runs the `config init` flow with the supplied IO.
// Tests use it instead of executing the cobra subcommand directly so
// the IsTTY/ReadPassword hooks can be substituted.
var RunConfigInit = runConfigInit

// RunConfigAddProfile, RunConfigUseProfile, RunConfigListProfiles are
// the test-visible entry points for the multi-profile management
// subcommands.
var (
	RunConfigAddProfile   = runConfigAddProfile
	RunConfigUseProfile   = runConfigUseProfile
	RunConfigListProfiles = runConfigListProfiles
)

// RevokeFunc mirrors the unexported revokeFunc dependency injected
// into runConfigUseProfile so tests can supply a spy.
type RevokeFunc = revokeFunc

// RevokeSession is the production revoke implementation, exposed so
// integration tests can drive the full soap.Decode + api.Call pipeline
// against an httptest.Server serving canned fixtures.
var RevokeSession = revokeSession

// RunSessionsDelete is the test-visible entry point for the
// `sessions delete` subcommand. Tests inject a RevokeFunc spy and a
// temp session.Store, mirroring the `config use-profile` pattern.
var RunSessionsDelete = runSessionsDelete

// DatabaseDeleteConfirm and MailAccountDeleteConfirm expose the
// package-private helpers that build the delete ConfirmAction for their
// slices. Tests use them to pin the "permanently delete" loudness
// adjustment — the two data-loss deletes (a database drops all rows, a
// mail account drops all stored messages) that use the emphatic verb
// instead of the bare "delete" every other slice uses.
var (
	DatabaseDeleteConfirm    = databaseDeleteConfirm
	MailAccountDeleteConfirm = mailAccountDeleteConfirm
)
