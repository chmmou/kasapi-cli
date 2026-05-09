package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/api"
	"github.com/chmmou/kasapi-cli/internal/directoryprotection"
)

// NewDirectoryProtectionCmd returns the "kasapi-cli directoryprotection"
// subcommand tree. The KAS endpoint `get_directoryprotection` returns
// one entry per (path, user) tuple, so a directory with N users
// surfaces as N rows; for that reason this resource is exposed as a
// list with an optional `--path` filter rather than the usual
// list+get pair.
func NewDirectoryProtectionCmd(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "directoryprotection",
		Short: "Inspect directory (htaccess) protections (get_directoryprotection)",
	}
	cmd.AddCommand(newDirectoryProtectionListCmd(opts))
	return cmd
}

func newDirectoryProtectionListCmd(opts *RootOptions) *cobra.Command {
	var path string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List directory protections, optionally filtered by --path",
		Args:  cobra.NoArgs,
		RunE: runListE(opts, "get_directoryprotection", func(c *api.Client, ctx context.Context) (directoryprotection.DirectoryProtectionList, error) {
			return directoryprotection.NewClient(c).List(ctx, path)
		}),
	}
	cmd.Flags().StringVar(&path, "path", "",
		"directory path to filter on (e.g. /protected/directory/); empty returns every protection")
	return cmd
}
