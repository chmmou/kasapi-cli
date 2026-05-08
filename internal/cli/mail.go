package cli

import (
	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/mailaccount"
)

// NewMailCmd returns the "kasapi-cli mail" subcommand tree, grouping
// the mail-related read endpoints (mail accounts; later forwards,
// filters, mailing lists).
func NewMailCmd(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mail",
		Short: "Inspect mail accounts, forwards, filters, and mailing lists",
	}
	cmd.AddCommand(newMailAccountsCmd(opts))
	return cmd
}

func newMailAccountsCmd(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accounts",
		Short: "Inspect mail accounts (get_mailaccounts)",
	}
	cmd.AddCommand(
		newMailAccountsListCmd(opts),
		newMailAccountsGetCmd(opts),
	)
	return cmd
}

func newMailAccountsListCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all mail accounts (get_mailaccounts)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := BuildAPIClient(opts)
			if err != nil {
				return err
			}
			list, err := mailaccount.NewClient(api).List(cmd.Context())
			if err != nil {
				return APIError(err, "get_mailaccounts")
			}
			if err := Render(cmd.OutOrStdout(), opts.Output, list); err != nil {
				return UserError(err, "render")
			}
			return nil
		},
	}
}

func newMailAccountsGetCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "get <mail-login>",
		Short: "Show details for a single mail account (get_mailaccounts with mail_login)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, err := BuildAPIClient(opts)
			if err != nil {
				return err
			}
			a, err := mailaccount.NewClient(api).Get(cmd.Context(), args[0])
			if err != nil {
				return APIError(err, "get_mailaccounts")
			}
			if err := Render(cmd.OutOrStdout(), opts.Output, a); err != nil {
				return UserError(err, "render")
			}
			return nil
		},
	}
}
