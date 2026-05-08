package cli

import (
	"github.com/spf13/cobra"

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
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := BuildAPIClient(opts)
			if err != nil {
				return err
			}
			list, err := softwareinstall.NewClient(api).List(cmd.Context())
			if err != nil {
				return APIError(err, "get_softwareinstall")
			}
			if err := Render(cmd.OutOrStdout(), opts.Output, list); err != nil {
				return UserError(err, "render")
			}
			return nil
		},
	}
}

func newSoftwareInstallsGetCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "get <software-id>",
		Short: "Show details for a single software template (get_softwareinstall with software_id)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := BuildAPIClient(opts)
			if err != nil {
				return err
			}
			s, err := softwareinstall.NewClient(api).Get(cmd.Context(), args[0])
			if err != nil {
				return APIError(err, "get_softwareinstall")
			}
			if err := Render(cmd.OutOrStdout(), opts.Output, s); err != nil {
				return UserError(err, "render")
			}
			return nil
		},
	}
}
