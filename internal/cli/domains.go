package cli

import (
	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/domain"
)

// NewDomainsCmd returns the "kasapi-cli domains" subcommand tree:
// list (get_domains), get (get_domains with domain_name).
func NewDomainsCmd(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domains",
		Short: "Inspect domains owned by the authenticated account",
	}
	cmd.AddCommand(
		newDomainsListCmd(opts),
		newDomainsGetCmd(opts),
	)
	return cmd
}

func newDomainsListCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all domains (get_domains)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := BuildAPIClient(opts)
			if err != nil {
				return err
			}
			list, err := domain.NewClient(api).List(cmd.Context())
			if err != nil {
				return APIError(err, "get_domains")
			}
			if err := Render(cmd.OutOrStdout(), opts.Output, list); err != nil {
				return UserError(err, "render")
			}
			return nil
		},
	}
}

func newDomainsGetCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "get <domain>",
		Short: "Show details for a single domain (get_domains with domain_name)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := BuildAPIClient(opts)
			if err != nil {
				return err
			}
			d, err := domain.NewClient(api).Get(cmd.Context(), args[0])
			if err != nil {
				return APIError(err, "get_domains")
			}
			if err := Render(cmd.OutOrStdout(), opts.Output, d); err != nil {
				return UserError(err, "render")
			}
			return nil
		},
	}
}
