package cli

import (
	"context"
	"errors"

	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/api"
	"github.com/chmmou/kasapi-cli/internal/dns"
)

// NewDNSCmd returns the "kasapi-cli dns" subcommand tree:
// list --domain <d> [--nameserver <ns>] (get_dns_settings).
func NewDNSCmd(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dns",
		Short: "Inspect DNS records for a zone",
	}
	cmd.AddCommand(newDNSListCmd(opts))
	return cmd
}

func newDNSListCmd(opts *RootOptions) *cobra.Command {
	var domainName, nameserver string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List DNS records for a zone (get_dns_settings)",
		Args:  cobra.NoArgs,
		// Validate the required flag in PreRunE so the missing-flag
		// path produces ExitUserError (1) rather than the
		// cli.CodeFor default of ExitAPIError (2). Cobra's own
		// MarkFlagRequired returns a plain error that would fall
		// through to the API-error code.
		PreRunE: func(_ *cobra.Command, _ []string) error {
			if domainName == "" {
				return UserError(errors.New("required flag --domain not provided"), "")
			}
			return nil
		},
		RunE: runListE(opts, "get_dns_settings", func(c *api.Client, ctx context.Context) (dns.RecordList, error) {
			return dns.NewClient(c).Settings(ctx, domainName, nameserver)
		}),
	}
	cmd.Flags().StringVar(&domainName, "domain", "", "zone host (required, e.g. example.com)")
	cmd.Flags().StringVar(&nameserver, "nameserver", "",
		"authoritative nameserver to query; empty uses the KAS default")
	return cmd
}
