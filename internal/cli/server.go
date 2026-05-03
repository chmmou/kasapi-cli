package cli

import (
	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/server"
)

// NewServerCmd returns the "kasapi-cli server" subcommand tree. For
// now it carries a single child, "info" (get_server_information).
func NewServerCmd(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Inspect the host server kasapi-cli is talking to",
	}
	cmd.AddCommand(newServerInfoCmd(opts))
	return cmd
}

func newServerInfoCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "List installed services and versions (get_server_information)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := BuildAPIClient(opts)
			if err != nil {
				return err
			}
			list, err := server.NewClient(api).Information(cmd.Context())
			if err != nil {
				return APIError(err, "get_server_information")
			}
			if err := Render(cmd.OutOrStdout(), opts.Output, list); err != nil {
				return UserError(err, "render")
			}
			return nil
		},
	}
}
