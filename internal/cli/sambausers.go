package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/api"
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
		RunE: runListE(opts, "get_sambausers", func(c *api.Client, ctx context.Context) (sambauser.SambaUserList, error) {
			return sambauser.NewClient(c).List(ctx)
		}),
	}
}

func newSambaUsersGetCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "get <samba-login>",
		Short: "Show details for a single Samba user (get_sambausers with samba_login)",
		Args:  cobra.ExactArgs(1),
		RunE: runGetE(opts, "get_sambausers", func(c *api.Client, ctx context.Context, arg string) (sambauser.SambaUser, error) {
			return sambauser.NewClient(c).Get(ctx, arg)
		}),
	}
}
