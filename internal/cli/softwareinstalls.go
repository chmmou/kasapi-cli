package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/api"
	"github.com/chmmou/kasapi-cli/internal/softwareinstall"
)

// NewSoftwareInstallsCmd returns the "kasapi-cli softwareinstalls"
// subcommand tree: list (get_softwareinstall, no filter) and get
// <software-id> (get_softwareinstall with a software_id filter).
// The KAS action is singular for both variants.
func NewSoftwareInstallsCmd(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "softwareinstalls",
		Short: "Inspect installable software templates (get_softwareinstall)",
	}
	cmd.AddCommand(
		newSoftwareInstallsListCmd(opts),
		newSoftwareInstallsGetCmd(opts),
	)
	return cmd
}

func newSoftwareInstallsListCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all installable software templates (get_softwareinstall, no filter)",
		Args:  cobra.NoArgs,
		RunE: runListE(opts, "get_softwareinstall", func(c *api.Client, ctx context.Context) (softwareinstall.SoftwareInstallList, error) {
			return softwareinstall.NewClient(c).List(ctx)
		}),
	}
}

func newSoftwareInstallsGetCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "get <software-id>",
		Short: "Show details for a single software template (get_softwareinstall with software_id)",
		Args:  cobra.ExactArgs(1),
		RunE: runGetE(opts, "get_softwareinstall", func(c *api.Client, ctx context.Context, arg string) (softwareinstall.SoftwareInstall, error) {
			return softwareinstall.NewClient(c).Get(ctx, arg)
		}),
	}
}
