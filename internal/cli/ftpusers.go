package cli

import (
	"github.com/spf13/cobra"

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
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := BuildAPIClient(opts)
			if err != nil {
				return err
			}
			list, err := ftpuser.NewClient(api).List(cmd.Context())
			if err != nil {
				return APIError(err, "get_ftpusers")
			}
			if err := Render(cmd.OutOrStdout(), opts.Output, list); err != nil {
				return UserError(err, "render")
			}
			return nil
		},
	}
}

func newFTPUsersGetCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "get <ftp-login>",
		Short: "Show details for a single FTP user (get_ftpusers with ftp_login)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := BuildAPIClient(opts)
			if err != nil {
				return err
			}
			u, err := ftpuser.NewClient(api).Get(cmd.Context(), args[0])
			if err != nil {
				return APIError(err, "get_ftpusers")
			}
			if err := Render(cmd.OutOrStdout(), opts.Output, u); err != nil {
				return UserError(err, "render")
			}
			return nil
		},
	}
}
