package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/api"
	"github.com/chmmou/kasapi-cli/internal/ddns"
)

// NewDDNSUsersCmd returns the "kasapi-cli ddnsusers" subcommand
// tree: list (get_ddnsusers, no filter) and get <dyndns-login>
// (get_ddnsusers with a ddns_login filter — note the wire-side
// asymmetry: the filter parameter has no `y`, unlike the response
// keys which use the dyndns_ prefix; per the KAS docs at
// get-ddnsusers-inc.html).
func NewDDNSUsersCmd(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ddnsusers",
		Short: "Inspect DDNS users visible to the login (get_ddnsusers)",
	}
	cmd.AddCommand(
		newDDNSUsersListCmd(opts),
		newDDNSUsersGetCmd(opts),
	)
	return cmd
}

func newDDNSUsersListCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all DDNS users (get_ddnsusers, no filter)",
		Args:  cobra.NoArgs,
		RunE: runListE(opts, "get_ddnsusers", func(c *api.Client, ctx context.Context) (ddns.DDNSUserList, error) {
			return ddns.NewClient(c).List(ctx)
		}),
	}
}

func newDDNSUsersGetCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "get <dyndns-login>",
		Short: "Show details for a single DDNS user (get_ddnsusers with ddns_login)",
		Args:  cobra.ExactArgs(1),
		RunE: runGetE(opts, "get_ddnsusers", func(c *api.Client, ctx context.Context, arg string) (ddns.DDNSUser, error) {
			return ddns.NewClient(c).Get(ctx, arg)
		}),
	}
}
