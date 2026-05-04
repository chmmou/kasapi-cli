package cli

import (
	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/domain"
)

// NewTLDsCmd returns the "kasapi-cli tlds" subcommand tree:
// list (get_topleveldomains).
func NewTLDsCmd(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tlds",
		Short: "Inspect the catalog of registrable top-level domains",
	}
	cmd.AddCommand(newTLDsListCmd(opts))
	return cmd
}

func newTLDsListCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all registrable TLDs (get_topleveldomains)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := BuildAPIClient(opts)
			if err != nil {
				return err
			}
			list, err := domain.NewClient(api).TopLevelDomains(cmd.Context())
			if err != nil {
				return APIError(err, "get_topleveldomains")
			}
			if err := Render(cmd.OutOrStdout(), opts.Output, list); err != nil {
				return UserError(err, "render")
			}
			return nil
		},
	}
}
