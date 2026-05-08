package cli

import (
	"github.com/spf13/cobra"

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
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := BuildAPIClient(opts)
			if err != nil {
				return err
			}
			list, err := ddns.NewClient(api).List(cmd.Context())
			if err != nil {
				return APIError(err, "get_ddnsusers")
			}
			if err := Render(cmd.OutOrStdout(), opts.Output, list); err != nil {
				return UserError(err, "render")
			}
			return nil
		},
	}
}

func newDDNSUsersGetCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "get <dyndns-login>",
		Short: "Show details for a single DDNS user (get_ddnsusers with ddns_login)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := BuildAPIClient(opts)
			if err != nil {
				return err
			}
			u, err := ddns.NewClient(api).Get(cmd.Context(), args[0])
			if err != nil {
				return APIError(err, "get_ddnsusers")
			}
			if err := Render(cmd.OutOrStdout(), opts.Output, u); err != nil {
				return UserError(err, "render")
			}
			return nil
		},
	}
}
