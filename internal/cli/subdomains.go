package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/api"
	"github.com/chmmou/kasapi-cli/internal/subdomain"
)

// NewSubdomainsCmd returns the "kasapi-cli subdomains" subcommand
// tree: list (get_subdomains), get (get_subdomains with subdomain_name).
func NewSubdomainsCmd(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subdomains",
		Short: "Inspect subdomains owned by the authenticated account",
	}
	cmd.AddCommand(
		newSubdomainsListCmd(opts),
		newSubdomainsGetCmd(opts),
	)
	return cmd
}

func newSubdomainsListCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all subdomains (get_subdomains)",
		Args:  cobra.NoArgs,
		RunE: runListE(opts, "get_subdomains", func(c *api.Client, ctx context.Context) (subdomain.SubdomainList, error) {
			return subdomain.NewClient(c).List(ctx)
		}),
	}
}

func newSubdomainsGetCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "get <subdomain-name>",
		Short: "Show details for a single subdomain (get_subdomains with subdomain_name)",
		Args:  cobra.ExactArgs(1),
		RunE: runGetE(opts, "get_subdomains", func(c *api.Client, ctx context.Context, arg string) (subdomain.Subdomain, error) {
			return subdomain.NewClient(c).Get(ctx, arg)
		}),
	}
}
