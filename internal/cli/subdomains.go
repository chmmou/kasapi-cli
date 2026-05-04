package cli

import (
	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/subdomain"
)

// NewSubdomainsCmd returns the "kasapi-cli subdomains" subcommand
// tree: list (get_subdomains), get (get_subdomains with subdomain_name).
func NewSubdomainsCmd(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subdomains",
		Short: "Inspect subdomains owned by the authenticated account",
	}
	cmd.AddCommand(
		newSubdomainsListCmd(opts),
		newSubdomainsGetCmd(opts),
	)
	return cmd
}

func newSubdomainsListCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all subdomains (get_subdomains)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := BuildAPIClient(opts)
			if err != nil {
				return err
			}
			list, err := subdomain.NewClient(api).List(cmd.Context())
			if err != nil {
				return APIError(err, "get_subdomains")
			}
			if err := Render(cmd.OutOrStdout(), opts.Output, list); err != nil {
				return UserError(err, "render")
			}
			return nil
		},
	}
}

func newSubdomainsGetCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "get <subdomain>",
		Short: "Show details for a single subdomain (get_subdomains with subdomain_name)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := BuildAPIClient(opts)
			if err != nil {
				return err
			}
			s, err := subdomain.NewClient(api).Get(cmd.Context(), args[0])
			if err != nil {
				return APIError(err, "get_subdomains")
			}
			if err := Render(cmd.OutOrStdout(), opts.Output, s); err != nil {
				return UserError(err, "render")
			}
			return nil
		},
	}
}
