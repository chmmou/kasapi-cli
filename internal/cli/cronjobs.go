package cli

import (
	"github.com/spf13/cobra"

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
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := BuildAPIClient(opts)
			if err != nil {
				return err
			}
			list, err := cronjob.NewClient(api).List(cmd.Context())
			if err != nil {
				return APIError(err, "get_cronjobs")
			}
			if err := Render(cmd.OutOrStdout(), opts.Output, list); err != nil {
				return UserError(err, "render")
			}
			return nil
		},
	}
}

func newCronjobsGetCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "get <cronjob-id>",
		Short: "Show details for a single cronjob (get_cronjobs with cronjob_id)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := BuildAPIClient(opts)
			if err != nil {
				return err
			}
			c, err := cronjob.NewClient(api).Get(cmd.Context(), args[0])
			if err != nil {
				return APIError(err, "get_cronjobs")
			}
			if err := Render(cmd.OutOrStdout(), opts.Output, c); err != nil {
				return UserError(err, "render")
			}
			return nil
		},
	}
}
