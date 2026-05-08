package cli

import (
	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/database"
)

// NewDatabasesCmd returns the "kasapi-cli databases" subcommand tree:
// list (get_databases, no filter) and get <database-login>
// (get_databases with a database_login filter).
func NewDatabasesCmd(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "databases",
		Short: "Inspect databases visible to the login (get_databases)",
	}
	cmd.AddCommand(
		newDatabasesListCmd(opts),
		newDatabasesGetCmd(opts),
	)
	return cmd
}

func newDatabasesListCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all databases (get_databases, no filter)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := BuildAPIClient(opts)
			if err != nil {
				return err
			}
			list, err := database.NewClient(api).List(cmd.Context())
			if err != nil {
				return APIError(err, "get_databases")
			}
			if err := Render(cmd.OutOrStdout(), opts.Output, list); err != nil {
				return UserError(err, "render")
			}
			return nil
		},
	}
}

func newDatabasesGetCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "get <database-login>",
		Short: "Show details for a single database (get_databases with database_login)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := BuildAPIClient(opts)
			if err != nil {
				return err
			}
			d, err := database.NewClient(api).Get(cmd.Context(), args[0])
			if err != nil {
				return APIError(err, "get_databases")
			}
			if err := Render(cmd.OutOrStdout(), opts.Output, d); err != nil {
				return UserError(err, "render")
			}
			return nil
		},
	}
}
