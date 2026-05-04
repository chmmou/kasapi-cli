package cli

import (
	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/usage"
)

// NewUsageCmd returns the "kasapi-cli usage" subcommand tree:
// space (get_space), space-detail (get_space_usage),
// traffic (get_traffic).
func NewUsageCmd(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "usage",
		Short: "Inspect webspace and traffic counters",
	}
	cmd.AddCommand(
		newUsageSpaceCmd(opts),
		newUsageSpaceDetailCmd(opts),
		newUsageTrafficCmd(opts),
	)
	return cmd
}

func newUsageSpaceCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "space",
		Short: "Show webspace totals per account (get_space)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := BuildAPIClient(opts)
			if err != nil {
				return err
			}
			list, err := usage.NewClient(api).Space(cmd.Context())
			if err != nil {
				return APIError(err, "get_space")
			}
			if err := Render(cmd.OutOrStdout(), opts.Output, list); err != nil {
				return UserError(err, "render")
			}
			return nil
		},
	}
}

func newUsageSpaceDetailCmd(opts *RootOptions) *cobra.Command {
	var directory string
	cmd := &cobra.Command{
		Use:   "space-detail",
		Short: "Show per-directory disk usage (get_space_usage)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := BuildAPIClient(opts)
			if err != nil {
				return err
			}
			list, err := usage.NewClient(api).SpaceUsage(cmd.Context(), directory)
			if err != nil {
				return APIError(err, "get_space_usage")
			}
			if err := Render(cmd.OutOrStdout(), opts.Output, list); err != nil {
				return UserError(err, "render")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&directory, "directory", "",
		"directory to drill into; empty queries the document-root level")
	return cmd
}

func newUsageTrafficCmd(opts *RootOptions) *cobra.Command {
	var year, month int
	cmd := &cobra.Command{
		Use:   "traffic",
		Short: "Show monthly HTTP/FTP traffic (get_traffic)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := BuildAPIClient(opts)
			if err != nil {
				return err
			}
			list, err := usage.NewClient(api).Traffic(cmd.Context(), year, month)
			if err != nil {
				return APIError(err, "get_traffic")
			}
			if err := Render(cmd.OutOrStdout(), opts.Output, list); err != nil {
				return UserError(err, "render")
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&year, "year", 0, "calendar year (e.g. 2026); 0 = current")
	cmd.Flags().IntVar(&month, "month", 0, "calendar month 1..12; 0 = current")
	return cmd
}
