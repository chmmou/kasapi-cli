package cli

import (
	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/sambauser"
)

// NewSambaUsersCmd returns the "kasapi-cli sambausers" subcommand
// tree: list (get_sambausers, no filter) and get <samba-login>
// (get_sambausers with a samba_login filter).
func NewSambaUsersCmd(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sambausers",
		Short: "Inspect Samba/CIFS users visible to the login (get_sambausers)",
	}
	cmd.AddCommand(
		newSambaUsersListCmd(opts),
		newSambaUsersGetCmd(opts),
	)
	return cmd
}

func newSambaUsersListCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all Samba users (get_sambausers, no filter)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := BuildAPIClient(opts)
			if err != nil {
				return err
			}
			list, err := sambauser.NewClient(api).List(cmd.Context())
			if err != nil {
				return APIError(err, "get_sambausers")
			}
			if err := Render(cmd.OutOrStdout(), opts.Output, list); err != nil {
				return UserError(err, "render")
			}
			return nil
		},
	}
}

func newSambaUsersGetCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "get <samba-login>",
		Short: "Show details for a single Samba user (get_sambausers with samba_login)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := BuildAPIClient(opts)
			if err != nil {
				return err
			}
			u, err := sambauser.NewClient(api).Get(cmd.Context(), args[0])
			if err != nil {
				return APIError(err, "get_sambausers")
			}
			if err := Render(cmd.OutOrStdout(), opts.Output, u); err != nil {
				return UserError(err, "render")
			}
			return nil
		},
	}
}
