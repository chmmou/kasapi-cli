package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/api"
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
		RunE: runListE(opts, "get_topleveldomains", func(c *api.Client, ctx context.Context) (domain.TLDList, error) {
			return domain.NewClient(c).TopLevelDomains(ctx)
		}),
	}
}
