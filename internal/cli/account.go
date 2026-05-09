package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/account"
	"github.com/chmmou/kasapi-cli/internal/api"
)

// NewAccountCmd returns the "kasapi-cli accounts" subcommand tree:
// list and get (both get_accounts, the latter with an account_login
// filter), settings (get_accountsettings), and resources
// (get_accountresources).
func NewAccountCmd(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accounts",
		Short: "Inspect KAS accounts owned by the authenticated login",
	}
	cmd.AddCommand(
		newAccountsListCmd(opts),
		newAccountsGetCmd(opts),
		newAccountsSettingsCmd(opts),
		newAccountsResourcesCmd(opts),
	)
	return cmd
}

func newAccountsListCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List accounts visible to the login (get_accounts, no filter)",
		Args:  cobra.NoArgs,
		RunE: runListE(opts, "get_accounts", func(c *api.Client, ctx context.Context) (account.AccountList, error) {
			accs, err := account.NewClient(c).List(ctx)
			return account.AccountList(accs), err
		}),
	}
}

func newAccountsGetCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "get <account-login>",
		Short: "Show details for a single account (get_accounts with account_login)",
		Args:  cobra.ExactArgs(1),
		RunE: runGetE(opts, "get_accounts", func(c *api.Client, ctx context.Context, arg string) (account.Account, error) {
			return account.NewClient(c).Get(ctx, arg)
		}),
	}
}

func newAccountsSettingsCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "settings",
		Short: "Show settings for the authenticated account (get_accountsettings)",
		Args:  cobra.NoArgs,
		RunE: runListE(opts, "get_accountsettings", func(c *api.Client, ctx context.Context) (account.AccountSettings, error) {
			return account.NewClient(c).Settings(ctx)
		}),
	}
}

func newAccountsResourcesCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "resources",
		Short: "Show quota counters for the authenticated account (get_accountresources)",
		Args:  cobra.NoArgs,
		RunE: runListE(opts, "get_accountresources", func(c *api.Client, ctx context.Context) (account.AccountResources, error) {
			return account.NewClient(c).Resources(ctx)
		}),
	}
}
