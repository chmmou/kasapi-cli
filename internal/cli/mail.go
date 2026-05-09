package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/api"
	"github.com/chmmou/kasapi-cli/internal/mailaccount"
	"github.com/chmmou/kasapi-cli/internal/mailfilter"
	"github.com/chmmou/kasapi-cli/internal/mailforward"
	"github.com/chmmou/kasapi-cli/internal/mailinglist"
)

// NewMailCmd returns the "kasapi-cli mail" subcommand tree, grouping
// the mail-related read endpoints (mail accounts; later forwards,
// filters, mailing lists).
func NewMailCmd(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mail",
		Short: "Inspect mail accounts, forwards, filters, and mailing lists",
	}
	cmd.AddCommand(
		newMailAccountsCmd(opts),
		newMailForwardsCmd(opts),
		newMailFiltersCmd(opts),
		newMailListsCmd(opts),
	)
	return cmd
}

func newMailListsCmd(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lists",
		Short: "Inspect mailing lists (get_mailinglists)",
	}
	cmd.AddCommand(
		newMailListsListCmd(opts),
		newMailListsGetCmd(opts),
	)
	return cmd
}

func newMailListsListCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all mailing lists (get_mailinglists)",
		Args:  cobra.NoArgs,
		RunE: runListE(opts, "get_mailinglists", func(c *api.Client, ctx context.Context) (mailinglist.MailingListList, error) {
			return mailinglist.NewClient(c).List(ctx)
		}),
	}
}

func newMailListsGetCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "get <mailinglist-name>",
		Short: "Show details for a single mailing list (get_mailinglists with mailinglist_name)",
		Args:  cobra.ExactArgs(1),
		RunE: runGetE(opts, "get_mailinglists", func(c *api.Client, ctx context.Context, arg string) (mailinglist.MailingList, error) {
			return mailinglist.NewClient(c).Get(ctx, arg)
		}),
	}
}

func newMailFiltersCmd(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "filters",
		Short: "Inspect mail standard filters (get_mailstandardfilter)",
	}
	cmd.AddCommand(newMailFiltersListCmd(opts))
	return cmd
}

func newMailFiltersListCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List the available standard mail filters (get_mailstandardfilter)",
		Args:  cobra.NoArgs,
		RunE: runListE(opts, "get_mailstandardfilter", func(c *api.Client, ctx context.Context) (mailfilter.StandardFilterList, error) {
			return mailfilter.NewClient(c).List(ctx)
		}),
	}
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
		RunE: runListE(opts, "get_mailaccounts", func(c *api.Client, ctx context.Context) (mailaccount.MailAccountList, error) {
			return mailaccount.NewClient(c).List(ctx)
		}),
	}
}

func newMailAccountsGetCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "get <mail-login>",
		Short: "Show details for a single mail account (get_mailaccounts with mail_login)",
		Args:  cobra.ExactArgs(1),
		RunE: runGetE(opts, "get_mailaccounts", func(c *api.Client, ctx context.Context, arg string) (mailaccount.MailAccount, error) {
			return mailaccount.NewClient(c).Get(ctx, arg)
		}),
	}
}

func newMailForwardsCmd(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "forwards",
		Short: "Inspect mail forwards (get_mailforwards)",
	}
	cmd.AddCommand(
		newMailForwardsListCmd(opts),
		newMailForwardsGetCmd(opts),
	)
	return cmd
}

func newMailForwardsListCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all mail forwards (get_mailforwards)",
		Args:  cobra.NoArgs,
		RunE: runListE(opts, "get_mailforwards", func(c *api.Client, ctx context.Context) (mailforward.MailForwardList, error) {
			return mailforward.NewClient(c).List(ctx)
		}),
	}
}

func newMailForwardsGetCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "get <address>",
		Short: "Show details for a single mail forward (get_mailforwards with mail_forward)",
		Args:  cobra.ExactArgs(1),
		RunE: runGetE(opts, "get_mailforwards", func(c *api.Client, ctx context.Context, arg string) (mailforward.MailForward, error) {
			return mailforward.NewClient(c).Get(ctx, arg)
		}),
	}
}
