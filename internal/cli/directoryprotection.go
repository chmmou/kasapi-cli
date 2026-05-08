package cli

import (
	"github.com/spf13/cobra"

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
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := BuildAPIClient(opts)
			if err != nil {
				return err
			}
			list, err := directoryprotection.NewClient(api).List(cmd.Context(), path)
			if err != nil {
				return APIError(err, "get_directoryprotection")
			}
			if err := Render(cmd.OutOrStdout(), opts.Output, list); err != nil {
				return UserError(err, "render")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&path, "path", "",
		"directory path to filter on (e.g. /protected/directory/); empty returns every protection")
	return cmd
}
