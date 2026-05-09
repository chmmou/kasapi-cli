package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/api"
	"github.com/chmmou/kasapi-cli/internal/ftpuser"
)

// NewFTPUsersCmd returns the "kasapi-cli ftpusers" subcommand tree:
// list (get_ftpusers, no filter) and get <ftp-login> (get_ftpusers
// with an ftp_login filter).
func NewFTPUsersCmd(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ftpusers",
		Short: "Inspect FTP users visible to the login (get_ftpusers)",
	}
	cmd.AddCommand(
		newFTPUsersListCmd(opts),
		newFTPUsersGetCmd(opts),
	)
	return cmd
}

func newFTPUsersListCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all FTP users (get_ftpusers, no filter)",
		Args:  cobra.NoArgs,
		RunE: runListE(opts, "get_ftpusers", func(c *api.Client, ctx context.Context) (ftpuser.FTPUserList, error) {
			return ftpuser.NewClient(c).List(ctx)
		}),
	}
}

func newFTPUsersGetCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "get <ftp-login>",
		Short: "Show details for a single FTP user (get_ftpusers with ftp_login)",
		Args:  cobra.ExactArgs(1),
		RunE: runGetE(opts, "get_ftpusers", func(c *api.Client, ctx context.Context, arg string) (ftpuser.FTPUser, error) {
			return ftpuser.NewClient(c).Get(ctx, arg)
		}),
	}
}
