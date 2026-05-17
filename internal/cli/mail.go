package cli

import (
	"context"
	"fmt"
	"strings"

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
		Short: "Inspect and manage mail forwards (get/add/update/delete_mailforward)",
	}
	cmd.AddCommand(
		newMailForwardsListCmd(opts),
		newMailForwardsGetCmd(opts),
		newMailForwardsAddCmd(opts),
		newMailForwardsUpdateCmd(opts),
		newMailForwardsDeleteCmd(opts),
	)
	return cmd
}

// splitMailAddress splits "local@domain" on the last '@' into the
// local_part / domain_part add_mailforward expects. get/update/delete
// take the full address verbatim; only add needs it decomposed.
func splitMailAddress(addr string) (local, domain string, err error) {
	at := strings.LastIndex(addr, "@")
	if at <= 0 || at == len(addr)-1 {
		return "", "", fmt.Errorf("invalid mail forward address %q: want local@domain", addr)
	}
	return addr[:at], addr[at+1:], nil
}

func newMailForwardsAddCmd(opts *RootOptions) *cobra.Command {
	var targets []string
	cmd := &cobra.Command{
		Use:   "add <address> --target <addr> [--target <addr>...]",
		Short: "Create a mail forward (add_mailforward)",
		Args:  cobra.ExactArgs(1),
		RunE: runWriteE(opts, func(args []string) (writeSpec, error) {
			local, domain, err := splitMailAddress(args[0])
			if err != nil {
				return writeSpec{}, err
			}
			if len(targets) == 0 {
				return writeSpec{}, fmt.Errorf("at least one --target is required")
			}
			return writeSpec{
				action:      "add_mailforward",
				destructive: false,
				confirm:     ConfirmAction{Verb: "create", Resource: "mail forward", ID: args[0]},
				params:      mailforward.AddParams(local, domain, targets),
				dispatch: func(c *api.Client, ctx context.Context) (string, error) {
					addr, derr := mailforward.NewClient(c).Add(ctx, local, domain, targets)
					if derr != nil {
						return "", derr
					}
					return "created mail forward " + addr, nil
				},
			}, nil
		}),
	}
	cmd.Flags().StringArrayVar(&targets, "target", nil, "forward target address (repeatable; replaces the full target list)")
	return cmd
}

func newMailForwardsUpdateCmd(opts *RootOptions) *cobra.Command {
	var targets []string
	cmd := &cobra.Command{
		Use:   "update <address> --target <addr> [--target <addr>...]",
		Short: "Replace the targets of a mail forward (update_mailforward)",
		Args:  cobra.ExactArgs(1),
		RunE: runWriteE(opts, func(args []string) (writeSpec, error) {
			if len(targets) == 0 {
				return writeSpec{}, fmt.Errorf("at least one --target is required")
			}
			address := args[0]
			return writeSpec{
				action:      "update_mailforward",
				destructive: true,
				confirm:     ConfirmAction{Verb: "replace the targets of", Resource: "mail forward", ID: address},
				params:      mailforward.UpdateParams(address, targets),
				dispatch: func(c *api.Client, ctx context.Context) (string, error) {
					if derr := mailforward.NewClient(c).Update(ctx, address, targets); derr != nil {
						return "", derr
					}
					return "updated mail forward " + address, nil
				},
			}, nil
		}),
	}
	cmd.Flags().StringArrayVar(&targets, "target", nil, "forward target address (repeatable; replaces the full target list)")
	return cmd
}

func newMailForwardsDeleteCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <address>",
		Short: "Delete a mail forward (delete_mailforward)",
		Args:  cobra.ExactArgs(1),
		RunE: runWriteE(opts, func(args []string) (writeSpec, error) {
			address := args[0]
			return writeSpec{
				action:      "delete_mailforward",
				destructive: true,
				confirm:     ConfirmAction{Verb: "delete", Resource: "mail forward", ID: address},
				params:      mailforward.DeleteParams(address),
				dispatch: func(c *api.Client, ctx context.Context) (string, error) {
					if derr := mailforward.NewClient(c).Delete(ctx, address); derr != nil {
						return "", derr
					}
					return "deleted mail forward " + address, nil
				},
			}, nil
		}),
	}
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
		Use:   "get <mail-forward>",
		Short: "Show details for a single mail forward (get_mailforwards with mail_forward)",
		Args:  cobra.ExactArgs(1),
		RunE: runGetE(opts, "get_mailforwards", func(c *api.Client, ctx context.Context, arg string) (mailforward.MailForward, error) {
			return mailforward.NewClient(c).Get(ctx, arg)
		}),
	}
}
