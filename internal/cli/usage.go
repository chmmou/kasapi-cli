package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/api"
	"github.com/chmmou/kasapi-cli/internal/usage"
)

// minTrafficYear is the lower bound for --year. KAS predates this date
// but the CLI is the kasapi-cli era, so 2000 is generous enough to be
// indistinguishable from "no validation" for any realistic input while
// still rejecting obvious typos like 200 or 20256.
const minTrafficYear = 2000

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
		RunE: runListE(opts, "get_space", func(c *api.Client, ctx context.Context) (usage.SpaceList, error) {
			return usage.NewClient(c).Space(ctx)
		}),
	}
}

func newUsageSpaceDetailCmd(opts *RootOptions) *cobra.Command {
	var directory string
	cmd := &cobra.Command{
		Use:   "space-detail",
		Short: "Show per-directory disk usage (get_space_usage)",
		Args:  cobra.NoArgs,
		RunE: runListE(opts, "get_space_usage", func(c *api.Client, ctx context.Context) (usage.SpaceUsageList, error) {
			return usage.NewClient(c).SpaceUsage(ctx, directory)
		}),
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
		PreRunE: func(_ *cobra.Command, _ []string) error {
			maxYear := time.Now().Year() + 1
			if year != 0 && (year < minTrafficYear || year > maxYear) {
				return UserError(fmt.Errorf("--year must be between %d and %d, got %d",
					minTrafficYear, maxYear, year), "")
			}
			if month != 0 && (month < 1 || month > 12) {
				return UserError(fmt.Errorf("--month must be between 1 and 12, got %d", month), "")
			}
			return nil
		},
		RunE: runListE(opts, "get_traffic", func(c *api.Client, ctx context.Context) (usage.TrafficList, error) {
			return usage.NewClient(c).Traffic(ctx, year, month)
		}),
	}
	cmd.Flags().IntVar(&year, "year", 0, "calendar year (e.g. 2026); 0 = current")
	cmd.Flags().IntVar(&month, "month", 0, "calendar month 1..12; 0 = current")
	return cmd
}
