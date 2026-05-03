package cli

// Internal hooks exported for tests in the cli_test package.

// ConfigIO mirrors the unexported configIO struct used by `config init`.
// Tests construct one to drive prompts and password input without a TTY.
type ConfigIO = configIO

// RunConfigInit runs the `config init` flow with the supplied IO.
// Tests use it instead of executing the cobra subcommand directly so
// the IsTTY/ReadPassword hooks can be substituted.
var RunConfigInit = runConfigInit
