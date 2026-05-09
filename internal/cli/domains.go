package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/api"
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
		RunE: runListE(opts, "get_domains", func(c *api.Client, ctx context.Context) (domain.DomainList, error) {
			return domain.NewClient(c).List(ctx)
		}),
	}
}

func newDomainsGetCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "get <domain>",
		Short: "Show details for a single domain (get_domains with domain_name)",
		Args:  cobra.ExactArgs(1),
		RunE: runGetE(opts, "get_domains", func(c *api.Client, ctx context.Context, arg string) (domain.Domain, error) {
			return domain.NewClient(c).Get(ctx, arg)
		}),
	}
}
