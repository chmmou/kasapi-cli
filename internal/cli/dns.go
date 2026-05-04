package cli

import (
	"errors"

	"github.com/spf13/cobra"

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
		RunE: func(cmd *cobra.Command, _ []string) error {
			if domainName == "" {
				return UserError(errors.New("required flag --domain not provided"), "--domain")
			}
			api, err := BuildAPIClient(opts)
			if err != nil {
				return err
			}
			list, err := dns.NewClient(api).Settings(cmd.Context(), domainName, nameserver)
			if err != nil {
				return APIError(err, "get_dns_settings")
			}
			if err := Render(cmd.OutOrStdout(), opts.Output, list); err != nil {
				return UserError(err, "render")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&domainName, "domain", "", "zone host (required, e.g. example.com)")
	cmd.Flags().StringVar(&nameserver, "nameserver", "",
		"authoritative nameserver to query; empty uses the KAS default")
	return cmd
}
