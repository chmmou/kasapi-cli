package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/api"
	"github.com/chmmou/kasapi-cli/internal/cronjob"
)

// NewCronjobsCmd returns the "kasapi-cli cronjobs" subcommand tree:
// list (get_cronjobs, no filter) and get <cronjob-id> (get_cronjobs
// with a cronjob_id filter).
func NewCronjobsCmd(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cronjobs",
		Short: "Inspect cronjobs visible to the login (get_cronjobs)",
	}
	cmd.AddCommand(
		newCronjobsListCmd(opts),
		newCronjobsGetCmd(opts),
	)
	return cmd
}

func newCronjobsListCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all cronjobs (get_cronjobs, no filter)",
		Args:  cobra.NoArgs,
		RunE: runListE(opts, "get_cronjobs", func(c *api.Client, ctx context.Context) (cronjob.CronjobList, error) {
			return cronjob.NewClient(c).List(ctx)
		}),
	}
}

func newCronjobsGetCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "get <cronjob-id>",
		Short: "Show details for a single cronjob (get_cronjobs with cronjob_id)",
		Args:  cobra.ExactArgs(1),
		RunE: runGetE(opts, "get_cronjobs", func(c *api.Client, ctx context.Context, arg string) (cronjob.Cronjob, error) {
			return cronjob.NewClient(c).Get(ctx, arg)
		}),
	}
}
