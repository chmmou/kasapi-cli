package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/version"
)

// RootOptions captures the values bound to the root command's
// persistent flags. Subcommands read it to obtain credentials,
// configuration paths, and the chosen output format.
//
// Output is parsed from the raw --output string in PersistentPreRunE so
// invalid values are rejected before any subcommand runs.
type RootOptions struct {
	ConfigPath string
	Profile    string
	Login      string
	AuthData   string
	AuthType   string
	OTP        string

	SessionLifetime       int
	SessionUpdateLifetime string

	OutputRaw string
	Output    Format

	NoColor bool
	Verbose bool
	Yes     bool
}

// NewRootCmd builds the kasapi-cli root command and registers all
// persistent global flags on it. Subcommands are registered by callers
// (cmd/kasapi-cli) so the cli package does not import every resource
// package directly.
//
// The returned *RootOptions is updated in place by cobra as flags are
// parsed; subcommands receive it through closure capture or by reading
// it back from cmd.Context().
func NewRootCmd() (*cobra.Command, *RootOptions) {
	opts := &RootOptions{}
	cmd := &cobra.Command{
		Use:           "kasapi-cli",
		Short:         "Command-line client for the All-Inkl KAS API",
		Long:          "kasapi-cli is a command-line client for the All-Inkl Kunden-Administrations-System SOAP API.",
		Version:       version.String(),
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			f, err := ParseFormat(opts.OutputRaw)
			if err != nil {
				return UserError(err, "")
			}
			opts.Output = f
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.SetVersionTemplate("{{.Version}}\n")
	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return UserError(err, "")
	})

	pf := cmd.PersistentFlags()
	pf.StringVar(&opts.ConfigPath, "config", "", "path to the kasapi-cli config file (overrides the default location)")
	pf.StringVar(&opts.Profile, "profile", "", "profile to select from the config file (overrides default_profile)")
	pf.StringVar(&opts.Login, "login", "", "KAS login (overrides config and KAS_LOGIN)")
	pf.StringVar(&opts.AuthData, "auth-data", "", "KAS auth data (overrides config and KAS_AUTHDATA)")
	pf.StringVar(&opts.AuthType, "auth-type", "",
		"KAS auth strategy: 'plain' = send password on each KasApi call (no KasAuth, no 2FA support); "+
			"'session' = bootstrap via KasAuth and reuse the credential token. Overrides config and KAS_AUTHTYPE.")
	pf.StringVar(&opts.OTP, "otp", "",
		"2FA one-time PIN — sent to KasAuth as session_2fa during the credential-token bootstrap. "+
			"Requires auth_type=session; the KAS API does not document 2FA on direct kas_auth_type=plain calls.")
	pf.IntVar(&opts.SessionLifetime, "session-lifetime", 0,
		"session_lifetime in seconds passed to KasAuth (1..30000); 0 keeps the server default. "+
			"Requires auth_type=session.")
	pf.StringVar(&opts.SessionUpdateLifetime, "session-update-lifetime", "",
		"session_update_lifetime passed to KasAuth ('Y' = sliding window, 'N' = fixed). "+
			"Empty omits the parameter. Requires auth_type=session.")
	pf.StringVarP(&opts.OutputRaw, "output", "o", "", fmt.Sprintf("output format: %s (default %s)", joinFormats(), DefaultFormat))
	pf.BoolVar(&opts.NoColor, "no-color", false, "disable coloured output")
	pf.BoolVarP(&opts.Verbose, "verbose", "v", false, "enable verbose logging on stderr")
	pf.BoolVarP(&opts.Yes, "yes", "y", false, "skip confirmation prompts on destructive operations")

	// --no-color and --yes are reserved but not yet wired: no colourised
	// output is emitted today, and the destructive-write confirmation
	// gate is tracked in issue #109. Keep them parseable (so a future
	// release can honour them without a breaking flag re-introduction)
	// but hidden, so --help and the generated docs do not advertise
	// flags that currently do nothing.
	_ = pf.MarkHidden("no-color")
	_ = pf.MarkHidden("yes")

	return cmd, opts
}

func joinFormats() string {
	return strings.Join(formatNames, "|")
}
