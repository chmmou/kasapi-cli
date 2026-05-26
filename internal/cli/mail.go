package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/api"
	"github.com/chmmou/kasapi-cli/internal/mailaccount"
	"github.com/chmmou/kasapi-cli/internal/mailfilter"
	"github.com/chmmou/kasapi-cli/internal/mailforward"
	"github.com/chmmou/kasapi-cli/internal/mailinglist"
)

// NewMailCmd returns the "kasapi-cli mail" subcommand tree: accounts
// and standard filters are read-only (get_mailaccounts /
// get_mailstandardfilter), while forwards and mailing lists are read
// plus the add/update/delete write endpoints.
func NewMailCmd(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mail",
		Short: "Inspect mail accounts and filters; inspect and manage forwards and mailing lists",
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
		Short: "Inspect and manage mailing lists (get/add/update/delete_mailinglist)",
	}
	cmd.AddCommand(
		newMailListsListCmd(opts),
		newMailListsGetCmd(opts),
		newMailListsAddCmd(opts),
		newMailListsUpdateCmd(opts),
		newMailListsDeleteCmd(opts),
	)
	return cmd
}

func newMailListsAddCmd(opts *RootOptions) *cobra.Command {
	var domain, password string
	cmd := &cobra.Command{
		Use:   "add <name> --domain <domain> --password <pw>",
		Short: "Create a mailing list (add_mailinglist)",
		Args:  cobra.ExactArgs(1),
		RunE: runWriteE(opts, func(args []string) (writeSpec, error) {
			name := args[0]
			if domain == "" {
				return writeSpec{}, fmt.Errorf("--domain is required")
			}
			if password == "" {
				return writeSpec{}, fmt.Errorf("--password is required")
			}
			return writeSpec{
				action:      "add_mailinglist",
				destructive: false,
				confirm:     ConfirmAction{Verb: "create", Resource: "mailing list", ID: name},
				params:      mailinglist.AddParams(name, domain, password),
				dispatch: func(c *api.Client, ctx context.Context) (string, error) {
					id, derr := mailinglist.NewClient(c).Add(ctx, name, domain, password)
					if derr != nil {
						return "", derr
					}
					return "created mailing list " + id, nil
				},
			}, nil
		}),
	}
	cmd.Flags().StringVar(&domain, "domain", "", "the list's domain (required)")
	cmd.Flags().StringVar(&password, "password", "", "the list password (required)")
	return cmd
}

func newMailListsUpdateCmd(opts *RootOptions) *cobra.Command {
	var subscriber, restrictPost []string
	var configFile, active string
	cmd := &cobra.Command{
		Use:   "update <name> [--subscriber <addr>...] [--restrict-post <addr>...] [--config-file <path>] [--active Y|N]",
		Short: "Replace mutable fields of a mailing list (update_mailinglist)",
		Args:  cobra.ExactArgs(1),
	}
	cmd.RunE = runWriteE(opts, func(args []string) (writeSpec, error) {
		name := args[0]
		// Only the explicitly-set flags are sent: each field is a
		// wholesale replacement and an empty value is a meaningful
		// "clear", so presence is keyed on Flags().Changed, not on
		// the value being non-empty.
		fields := map[string]string{}
		if cmd.Flags().Changed("subscriber") {
			fields[mailinglist.FieldSubscriber] = strings.Join(subscriber, "\n")
		}
		if cmd.Flags().Changed("restrict-post") {
			fields[mailinglist.FieldRestrictPost] = strings.Join(restrictPost, "\n")
		}
		if cmd.Flags().Changed("config-file") {
			//nolint:gosec // G304: configFile is the explicit --config-file CLI argument; reading the user-named file is the documented intent.
			b, rerr := os.ReadFile(configFile)
			if rerr != nil {
				return writeSpec{}, fmt.Errorf("read --config-file: %w", rerr)
			}
			fields[mailinglist.FieldConfig] = string(b)
		}
		if cmd.Flags().Changed("active") {
			fields[mailinglist.FieldIsActive] = active
		}
		if len(fields) == 0 {
			return writeSpec{}, fmt.Errorf("at least one of --subscriber/--restrict-post/--config-file/--active is required")
		}
		return writeSpec{
			action:      "update_mailinglist",
			destructive: true,
			confirm:     ConfirmAction{Verb: "replace the settings of", Resource: "mailing list", ID: name},
			params:      mailinglist.UpdateParams(name, fields),
			dispatch: func(c *api.Client, ctx context.Context) (string, error) {
				if derr := mailinglist.NewClient(c).Update(ctx, name, fields); derr != nil {
					return "", derr
				}
				return "updated mailing list " + name, nil
			},
		}, nil
	})
	cmd.Flags().StringArrayVar(&subscriber, "subscriber", nil, "list subscriber address (repeatable; replaces the full subscriber list)")
	cmd.Flags().StringArrayVar(&restrictPost, "restrict-post", nil, "restrict-post address (repeatable; replaces the full restrict-post list)")
	cmd.Flags().StringVar(&configFile, "config-file", "", "path to the complete list configuration file (replaces the config wholesale)")
	cmd.Flags().StringVar(&active, "active", "", "list status (Y|N)")
	return cmd
}

func newMailListsDeleteCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a mailing list (delete_mailinglist)",
		Args:  cobra.ExactArgs(1),
		RunE: runWriteE(opts, func(args []string) (writeSpec, error) {
			name := args[0]
			return writeSpec{
				action:      "delete_mailinglist",
				destructive: true,
				confirm:     ConfirmAction{Verb: "delete", Resource: "mailing list", ID: name},
				params:      mailinglist.DeleteParams(name),
				dispatch: func(c *api.Client, ctx context.Context) (string, error) {
					if derr := mailinglist.NewClient(c).Delete(ctx, name); derr != nil {
						return "", derr
					}
					return "deleted mailing list " + name, nil
				},
			}, nil
		}),
	}
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
		Short: "Inspect and manage mail standard filters (get/add/delete_mailstandardfilter)",
	}
	cmd.AddCommand(
		newMailFiltersListCmd(opts),
		newMailFiltersAddCmd(opts),
		newMailFiltersDeleteCmd(opts),
	)
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

func newMailFiltersAddCmd(opts *RootOptions) *cobra.Command {
	var items []string
	cmd := &cobra.Command{
		Use:   "add <mail-login> --filter <item> [--filter <item>...]",
		Short: "Set the standard-filter chain on a mail account (add_mailstandardfilter)",
		Long: `Set the configured standard-filter chain on a mail account via
add_mailstandardfilter. Each --filter is one item of the chain, either a
bare filter id (e.g. "pdw") or "<filter-id>:<option>=<value>" (e.g.
"spamc_move:move=Spam"). Items are joined with ';' on the wire and the
chain replaces what was configured before — there is no per-item add.
Use "mail filters list" for the available filter ids.`,
		Args: cobra.ExactArgs(1),
		RunE: runWriteE(opts, func(args []string) (writeSpec, error) {
			if len(items) == 0 {
				return writeSpec{}, fmt.Errorf("at least one --filter is required")
			}
			// Validate + join here, before the writeSpec is built, so the
			// dry-run preview and the audit record see the exact chain
			// string that will be dispatched and an invalid item cannot
			// reach the audit log with a silently-empty filter field.
			chain, err := mailfilter.JoinFilters(items)
			if err != nil {
				return writeSpec{}, err
			}
			login := args[0]
			return writeSpec{
				action:      "add_mailstandardfilter",
				destructive: true,
				confirm:     ConfirmAction{Verb: "replace the standard-filter chain of", Resource: "mail account", ID: login},
				params:      mailfilter.AddParams(login, chain),
				dispatch: func(c *api.Client, ctx context.Context) (string, error) {
					if derr := mailfilter.NewClient(c).Add(ctx, login, items); derr != nil {
						return "", derr
					}
					return "configured mail standard filter for " + login, nil
				},
			}, nil
		}),
	}
	cmd.Flags().StringArrayVar(&items, "filter", nil, "filter chain item (repeatable; replaces the full chain)")
	return cmd
}

func newMailFiltersDeleteCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <mail-login>",
		Short: "Remove every standard filter from a mail account (delete_mailstandardfilter)",
		Long: `Remove every configured standard filter from a mail account via
delete_mailstandardfilter. The KAS action takes only the mail login and
drops the whole chain in one shot — there is no per-item delete.

Note: the API sometimes surfaces an envelope-level SOAP fault (an
internal "sizeof()" PHP error) even when the chain is in fact removed
on the server. The fault is surfaced verbatim; if you see it, verify
the actual outcome with "mail accounts get <login>" — the configured
chain is reported in the mail_spamfilter field.`,
		Args: cobra.ExactArgs(1),
		RunE: runWriteE(opts, func(args []string) (writeSpec, error) {
			login := args[0]
			return writeSpec{
				action:      "delete_mailstandardfilter",
				destructive: true,
				confirm:     ConfirmAction{Verb: "remove all standard filters of", Resource: "mail account", ID: login},
				params:      mailfilter.DeleteParams(login),
				dispatch: func(c *api.Client, ctx context.Context) (string, error) {
					if derr := mailfilter.NewClient(c).Delete(ctx, login); derr != nil {
						return "", derr
					}
					return "removed mail standard filters for " + login, nil
				},
			}, nil
		}),
	}
}

func newMailAccountsCmd(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accounts",
		Short: "Inspect and manage mail accounts (get/add/update/delete_mailaccount)",
	}
	cmd.AddCommand(
		newMailAccountsListCmd(opts),
		newMailAccountsGetCmd(opts),
		newMailAccountsAddCmd(opts),
		newMailAccountsUpdateCmd(opts),
		newMailAccountsDeleteCmd(opts),
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

// mailAccountAddFlags binds the add_mailaccount request fields. add and
// update bind disjoint flag sets (same wire-key surface, but each
// subcommand's --help reflects only its own action semantics, and cobra
// rejects a wrong-subcommand flag at parse time). The password flag maps
// to mail_password here (the add-only key); update has its own
// --password flag that maps to mail_new_password.
//
// The Y/N/text toggles and the XLIST folder names default to the values
// the KAS API documents as its own defaults (and that the captured
// add_mailaccount request fixture carries), so a bare
// "accounts add <addr> --password <pw>" produces a valid create.
type mailAccountAddFlags struct {
	password             string
	webmailAutologin     string
	responder            string
	responderContentType string
	responderDisplayName string
	responderText        string
	copyAddress          string
	senderAlias          string
	xlistEnabled         string
	xlistSent            string
	xlistDrafts          string
	xlistTrash           string
	xlistSpam            string
	xlistArchiv          string
	allowNets            string
}

func (f *mailAccountAddFlags) bind(cmd *cobra.Command) {
	fl := cmd.Flags()
	fl.StringVar(&f.password, "password", "", "initial mailbox password (required)")
	fl.StringVar(&f.webmailAutologin, "webmail-autologin", "Y", "allow KAS-to-webmail auto-login (Y|N)")
	fl.StringVar(&f.responder, "responder", "N", `auto-responder: "N", "Y", or a "<start>|<end>" timestamp range`)
	fl.StringVar(&f.responderContentType, "responder-content-type", "text", "auto-responder body format (html|text)")
	fl.StringVar(&f.responderDisplayName, "responder-displayname", "", "auto-responder sender display name")
	fl.StringVar(&f.responderText, "responder-text", "", "auto-responder message body")
	fl.StringVar(&f.copyAddress, "copy-address", "", "BCC copy recipient address(es)")
	fl.StringVar(&f.senderAlias, "sender-alias", "", "permitted FROM alias address(es)")
	fl.StringVar(&f.xlistEnabled, "xlist-enabled", "Y", "enable XLIST special-folder mapping (Y|N)")
	fl.StringVar(&f.xlistSent, "xlist-sent", "Sent", "XLIST sent-items folder name")
	fl.StringVar(&f.xlistDrafts, "xlist-drafts", "Drafts", "XLIST drafts folder name")
	fl.StringVar(&f.xlistTrash, "xlist-trash", "Trash", "XLIST trash folder name")
	fl.StringVar(&f.xlistSpam, "xlist-spam", "Spam", "XLIST spam folder name")
	fl.StringVar(&f.xlistArchiv, "xlist-archiv", "Archive", "XLIST archive folder name")
	fl.StringVar(&f.allowNets, "allow-nets", "", "restrict access to these IP/CIDR networks (empty = no restriction)")
}

func (f *mailAccountAddFlags) spec(local, domain string) mailaccount.Spec {
	return mailaccount.Spec{
		LocalPart:            local,
		DomainPart:           domain,
		Password:             f.password,
		WebmailAutologin:     f.webmailAutologin,
		Responder:            f.responder,
		ResponderContentType: f.responderContentType,
		ResponderDisplayName: f.responderDisplayName,
		ResponderText:        f.responderText,
		CopyAddress:          f.copyAddress,
		SenderAlias:          f.senderAlias,
		XListEnabled:         f.xlistEnabled,
		XListSent:            f.xlistSent,
		XListDrafts:          f.xlistDrafts,
		XListTrash:           f.xlistTrash,
		XListSpam:            f.xlistSpam,
		XListArchiv:          f.xlistArchiv,
		AllowNets:            f.allowNets,
	}
}

func newMailAccountsAddCmd(opts *RootOptions) *cobra.Command {
	f := &mailAccountAddFlags{}
	cmd := &cobra.Command{
		Use:   "add <address> --password <pw> [field flags]",
		Short: "Create a mail account (add_mailaccount; the login is generated by KAS)",
		Long: `Create a mail account via add_mailaccount. The address is split on the
last '@' into the local_part / domain_part KAS expects, and KAS
generates the mail login (e.g. m0000001), which the command prints on
success.

The Y/N/text toggles and XLIST folder names default to the KAS API's
own defaults, so "accounts add info@example.com --password <pw>" is a
complete create; override any field with its flag.`,
		Args: cobra.ExactArgs(1),
		RunE: runWriteE(opts, func(args []string) (writeSpec, error) {
			local, domain, err := splitMailAddress(args[0])
			if err != nil {
				return writeSpec{}, err
			}
			if f.password == "" {
				return writeSpec{}, fmt.Errorf("--password is required")
			}
			s := f.spec(local, domain)
			return writeSpec{
				action:      "add_mailaccount",
				destructive: false,
				confirm:     ConfirmAction{Verb: "create", Resource: "mail account", ID: args[0]},
				params:      mailaccount.AddParams(s),
				dispatch: func(c *api.Client, ctx context.Context) (string, error) {
					login, derr := mailaccount.NewClient(c).Add(ctx, s)
					if derr != nil {
						return "", derr
					}
					return "created mail account " + login, nil
				},
			}, nil
		}),
	}
	f.bind(cmd)
	return cmd
}

// mailAccountUpdateFlags binds the update_mailaccount mutable surface.
// Disjoint from mailAccountAddFlags: update has no local/domain (the
// account is addressed by its generated mail_login), adds the --active
// toggle (is_active), and its --password maps to mail_new_password (the
// update key) rather than the add-only mail_password. No flag carries a
// default — only the flags the user explicitly sets are sent, so an
// unset flag means "leave unchanged".
type mailAccountUpdateFlags struct {
	password             string
	active               string
	webmailAutologin     string
	responder            string
	responderContentType string
	responderDisplayName string
	responderText        string
	copyAddress          string
	senderAlias          string
	xlistEnabled         string
	xlistSent            string
	xlistDrafts          string
	xlistTrash           string
	xlistSpam            string
	xlistArchiv          string
	allowNets            string
}

func (f *mailAccountUpdateFlags) bind(cmd *cobra.Command) {
	fl := cmd.Flags()
	fl.StringVar(&f.password, "password", "", "replacement mailbox password (sent as mail_new_password)")
	fl.StringVar(&f.active, "active", "", "mailbox status (Y|N)")
	fl.StringVar(&f.webmailAutologin, "webmail-autologin", "", "allow KAS-to-webmail auto-login (Y|N)")
	fl.StringVar(&f.responder, "responder", "", `auto-responder: "N", "Y", or a "<start>|<end>" timestamp range`)
	fl.StringVar(&f.responderContentType, "responder-content-type", "", "auto-responder body format (html|text)")
	fl.StringVar(&f.responderDisplayName, "responder-displayname", "", "auto-responder sender display name")
	fl.StringVar(&f.responderText, "responder-text", "", "auto-responder message body")
	fl.StringVar(&f.copyAddress, "copy-address", "", "BCC copy recipient address(es)")
	fl.StringVar(&f.senderAlias, "sender-alias", "", "permitted FROM alias address(es)")
	fl.StringVar(&f.xlistEnabled, "xlist-enabled", "", "enable XLIST special-folder mapping (Y|N)")
	fl.StringVar(&f.xlistSent, "xlist-sent", "", "XLIST sent-items folder name")
	fl.StringVar(&f.xlistDrafts, "xlist-drafts", "", "XLIST drafts folder name")
	fl.StringVar(&f.xlistTrash, "xlist-trash", "", "XLIST trash folder name")
	fl.StringVar(&f.xlistSpam, "xlist-spam", "", "XLIST spam folder name")
	fl.StringVar(&f.xlistArchiv, "xlist-archiv", "", "XLIST archive folder name")
	fl.StringVar(&f.allowNets, "allow-nets", "", "restrict access to these IP/CIDR networks")
}

// mailAccountUpdateChangedFields collects only the flags the user
// explicitly set into the update_mailaccount field map (keyed on the
// mailaccount.Field* constants). Each field is a wholesale replacement
// and an empty value is a meaningful set, so presence is keyed on cobra
// Changed, not on the value being non-empty — the same pattern the
// database/ftpuser updates use. The password flag maps to
// mail_new_password here (update_mailaccount's key).
func mailAccountUpdateChangedFields(cmd *cobra.Command, f *mailAccountUpdateFlags) map[string]string {
	fields := map[string]string{}
	for _, m := range []struct {
		flag  string
		key   string
		value string
	}{
		{"password", mailaccount.FieldNewPassword, f.password},
		{"active", mailaccount.FieldIsActive, f.active},
		{"webmail-autologin", mailaccount.FieldWebmailAutologin, f.webmailAutologin},
		{"responder", mailaccount.FieldResponder, f.responder},
		{"responder-content-type", mailaccount.FieldResponderContentType, f.responderContentType},
		{"responder-displayname", mailaccount.FieldResponderDisplayName, f.responderDisplayName},
		{"responder-text", mailaccount.FieldResponderText, f.responderText},
		{"copy-address", mailaccount.FieldCopyAddress, f.copyAddress},
		{"sender-alias", mailaccount.FieldSenderAlias, f.senderAlias},
		{"xlist-enabled", mailaccount.FieldXListEnabled, f.xlistEnabled},
		{"xlist-sent", mailaccount.FieldXListSent, f.xlistSent},
		{"xlist-drafts", mailaccount.FieldXListDrafts, f.xlistDrafts},
		{"xlist-trash", mailaccount.FieldXListTrash, f.xlistTrash},
		{"xlist-spam", mailaccount.FieldXListSpam, f.xlistSpam},
		{"xlist-archiv", mailaccount.FieldXListArchiv, f.xlistArchiv},
		{"allow-nets", mailaccount.FieldAllowNets, f.allowNets},
	} {
		if cmd.Flags().Changed(m.flag) {
			fields[m.key] = m.value
		}
	}
	return fields
}

func newMailAccountsUpdateCmd(opts *RootOptions) *cobra.Command {
	f := &mailAccountUpdateFlags{}
	cmd := &cobra.Command{
		Use:   "update <mail-login> [field flags]",
		Short: "Replace mutable fields of a mail account (update_mailaccount)",
		Args:  cobra.ExactArgs(1),
	}
	cmd.RunE = runWriteE(opts, func(args []string) (writeSpec, error) {
		login := args[0]
		fields := mailAccountUpdateChangedFields(cmd, f)
		if len(fields) == 0 {
			return writeSpec{}, fmt.Errorf("at least one field flag (e.g. --password/--active/--responder) is required")
		}
		return writeSpec{
			action:      "update_mailaccount",
			destructive: true,
			confirm:     ConfirmAction{Verb: "replace the settings of", Resource: "mail account", ID: login},
			params:      mailaccount.UpdateParams(login, fields),
			dispatch: func(c *api.Client, ctx context.Context) (string, error) {
				if derr := mailaccount.NewClient(c).Update(ctx, login, fields); derr != nil {
					return "", derr
				}
				return "updated mail account " + login, nil
			},
		}, nil
	})
	f.bind(cmd)
	return cmd
}

// mailAccountDeleteConfirm builds the ConfirmAction shown before
// delete_mailaccount is dispatched. Like delete_database it uses the
// louder "permanently delete" verb: deleting a mail account drops the
// mailbox and every message stored in it, so this is data loss, not
// just metadata removal. The prompt template adds "This cannot be
// undone." regardless of the verb.
func mailAccountDeleteConfirm(login string) ConfirmAction {
	return ConfirmAction{Verb: "permanently delete", Resource: "mail account", ID: login}
}

func newMailAccountsDeleteCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <mail-login>",
		Short: "Permanently delete a mail account and all messages in it (delete_mailaccount)",
		Args:  cobra.ExactArgs(1),
		RunE: runWriteE(opts, func(args []string) (writeSpec, error) {
			login := args[0]
			return writeSpec{
				action:      "delete_mailaccount",
				destructive: true,
				confirm:     mailAccountDeleteConfirm(login),
				params:      mailaccount.DeleteParams(login),
				dispatch: func(c *api.Client, ctx context.Context) (string, error) {
					if derr := mailaccount.NewClient(c).Delete(ctx, login); derr != nil {
						return "", derr
					}
					return "deleted mail account " + login, nil
				},
			}, nil
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
// local_part / domain_part the add_mailforward and add_mailaccount
// actions expect. Their get/update/delete counterparts take the full
// address (forwards) or the generated mail_login (accounts) verbatim;
// only the add path needs the address decomposed.
func splitMailAddress(addr string) (local, domain string, err error) {
	at := strings.LastIndex(addr, "@")
	if at <= 0 || at == len(addr)-1 {
		return "", "", fmt.Errorf("invalid mail address %q: want local@domain", addr)
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
	cmd.Flags().StringArrayVar(&targets, "target", nil, "forward target address (repeatable; at least one required)")
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
