package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/api"
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
		RunE: runListE(opts, "get_databases", func(c *api.Client, ctx context.Context) (database.DatabaseList, error) {
			return database.NewClient(c).List(ctx)
		}),
	}
}

func newDatabasesGetCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "get <database-login>",
		Short: "Show details for a single database (get_databases with database_login)",
		Args:  cobra.ExactArgs(1),
		RunE: runGetE(opts, "get_databases", func(c *api.Client, ctx context.Context, arg string) (database.Database, error) {
			return database.NewClient(c).Get(ctx, arg)
		}),
	}
}
