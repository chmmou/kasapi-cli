package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/api"
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
		RunE: runListE(opts, "get_server_information", func(c *api.Client, ctx context.Context) (server.ServiceList, error) {
			return server.NewClient(c).Information(ctx)
		}),
	}
}
