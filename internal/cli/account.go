package cli

import (
	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/account"
)

// NewAccountCmd returns the "kasapi-cli accounts" subcommand tree:
// list (get_accounts), get (get_accountsettings), and resources
// (get_accountresources).
func NewAccountCmd(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accounts",
		Short: "Inspect KAS accounts owned by the authenticated login",
	}
	cmd.AddCommand(
		newAccountsListCmd(opts),
		newAccountsGetCmd(opts),
		newAccountsResourcesCmd(opts),
	)
	return cmd
}

func newAccountsListCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all sub-accounts (get_accounts)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := BuildAPIClient(opts)
			if err != nil {
				return err
			}
			accs, err := account.NewClient(api).List(cmd.Context())
			if err != nil {
				return APIError(err, "get_accounts")
			}
			if err := Render(cmd.OutOrStdout(), opts.Output, account.AccountList(accs)); err != nil {
				return UserError(err, "render")
			}
			return nil
		},
	}
}

func newAccountsGetCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Show settings for the authenticated account (get_accountsettings)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := BuildAPIClient(opts)
			if err != nil {
				return err
			}
			s, err := account.NewClient(api).Settings(cmd.Context())
			if err != nil {
				return APIError(err, "get_accountsettings")
			}
			if err := Render(cmd.OutOrStdout(), opts.Output, s); err != nil {
				return UserError(err, "render")
			}
			return nil
		},
	}
}

func newAccountsResourcesCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "resources",
		Short: "Show quota counters for the authenticated account (get_accountresources)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := BuildAPIClient(opts)
			if err != nil {
				return err
			}
			r, err := account.NewClient(api).Resources(cmd.Context())
			if err != nil {
				return APIError(err, "get_accountresources")
			}
			if err := Render(cmd.OutOrStdout(), opts.Output, r); err != nil {
				return UserError(err, "render")
			}
			return nil
		},
	}
}
