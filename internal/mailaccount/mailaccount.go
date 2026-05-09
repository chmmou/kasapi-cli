package mailaccount

import (
	"context"
	"strconv"

	"github.com/chmmou/kasapi-cli/internal/kasread"
	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/tablefmt"
)

// Caller is the subset of *api.Client this package depends on. The
// indirection keeps tests free of network setup: a fake Caller can
// return a *soap.Response decoded from a fixture.
type Caller = kasread.Caller

// MailAccount is one entry of get_mailaccounts. The list and singular
// views (the latter being get_mailaccounts called with a mail_login
// filter) return the same Map shape, so a single struct covers both.
type MailAccount struct {
	Login    string `json:"mail_login" yaml:"mail_login"`
	Password string `json:"mail_password,omitempty" yaml:"mail_password,omitempty"`

	// The KAS API returns both the legacy mail_adresses key (single d)
	// and the canonical mail_addresses key with identical content. We
	// keep both in the struct so a JSON/YAML round-trip preserves the
	// raw payload, but consumers should prefer Addresses.
	Adresses  string `json:"mail_adresses" yaml:"mail_adresses"`
	Addresses string `json:"mail_addresses" yaml:"mail_addresses"`

	Comment string `json:"mail_comment" yaml:"mail_comment"`

	Responder            string `json:"mail_responder" yaml:"mail_responder"`
	ResponderText        string `json:"mail_responder_text" yaml:"mail_responder_text"`
	ResponderDisplayName string `json:"mail_responder_displayname" yaml:"mail_responder_displayname"`
	ResponderContentType string `json:"mail_responder_content_type" yaml:"mail_responder_content_type"`

	// Same legacy/canonical pair as for the addresses.
	CopyAdress  string `json:"mail_copy_adress" yaml:"mail_copy_adress"`
	CopyAddress string `json:"mail_copy_address" yaml:"mail_copy_address"`

	SenderAlias string `json:"mail_sender_alias" yaml:"mail_sender_alias"`
	Spamfilter  string `json:"mail_spamfilter" yaml:"mail_spamfilter"`

	InProgress string `json:"in_progress" yaml:"in_progress"`

	XListEnabled string `json:"mail_xlist_enabled" yaml:"mail_xlist_enabled"`
	XListSent    string `json:"mail_xlist_sent" yaml:"mail_xlist_sent"`
	XListDrafts  string `json:"mail_xlist_drafts" yaml:"mail_xlist_drafts"`
	XListTrash   string `json:"mail_xlist_trash" yaml:"mail_xlist_trash"`
	XListSpam    string `json:"mail_xlist_spam" yaml:"mail_xlist_spam"`
	XListArchiv  string `json:"mail_xlist_archiv" yaml:"mail_xlist_archiv"`

	UsedSpace        float64 `json:"used_mailaccount_space" yaml:"used_mailaccount_space"`
	IsActive         string  `json:"mail_is_active" yaml:"mail_is_active"`
	ShowPassword     string  `json:"show_password" yaml:"show_password"`
	AllowNets        string  `json:"mail_allow_nets" yaml:"mail_allow_nets"`
	TwoFA            string  `json:"mail_2fa" yaml:"mail_2fa"`
	QuotaRule        int     `json:"quota_rule" yaml:"quota_rule"`
	WebmailAutologin string  `json:"webmail_autologin" yaml:"webmail_autologin"`
}

// MailAccountList is the typed payload of get_mailaccounts; satisfies
// cli.Tabular.
type MailAccountList []MailAccount

// Client groups the read endpoints scoped to mail accounts:
// get_mailaccounts (list and singular).
type Client struct {
	lg kasread.ListGet[MailAccountList, MailAccount]
}

// NewClient returns a Client backed by the given Caller.
func NewClient(c Caller) *Client {
	return &Client{lg: kasread.ListGet[MailAccountList, MailAccount]{
		Caller:    c,
		Action:    "get_mailaccounts",
		Label:     "mailaccount",
		ArgName:   "login",
		FilterKey: "mail_login",
		Decoder:   DecodeMailAccounts,
	}}
}

// List calls get_mailaccounts without parameters and decodes the
// response into a MailAccountList covering every mail account visible
// to the login.
func (c *Client) List(ctx context.Context) (MailAccountList, error) { return c.lg.List(ctx) }

// Get calls get_mailaccounts with a mail_login filter and returns the
// single matching MailAccount. The KAS API still wraps the result in
// an array; we unwrap it here so callers do not have to. An empty
// array surfaces as a not-found error.
func (c *Client) Get(ctx context.Context, login string) (MailAccount, error) {
	return c.lg.Get(ctx, login)
}

// DecodeMailAccounts maps the ReturnInfo of a get_mailaccounts response
// (an Array of Maps) into the typed MailAccountList.
func DecodeMailAccounts(returnInfo soap.Value) (MailAccountList, error) {
	out, err := soap.DecodeArray(returnInfo, "mailaccount", func(item soap.Value) MailAccount {
		return MailAccount{
			Login:                item.MapString("mail_login"),
			Password:             item.MapString("mail_password"),
			Adresses:             item.MapString("mail_adresses"),
			Addresses:            item.MapString("mail_addresses"),
			Comment:              item.MapString("mail_comment"),
			Responder:            item.MapString("mail_responder"),
			ResponderText:        item.MapString("mail_responder_text"),
			ResponderDisplayName: item.MapString("mail_responder_displayname"),
			ResponderContentType: item.MapString("mail_responder_content_type"),
			CopyAdress:           item.MapString("mail_copy_adress"),
			CopyAddress:          item.MapString("mail_copy_address"),
			SenderAlias:          item.MapString("mail_sender_alias"),
			Spamfilter:           item.MapString("mail_spamfilter"),
			InProgress:           item.MapString("in_progress"),
			XListEnabled:         item.MapString("mail_xlist_enabled"),
			XListSent:            item.MapString("mail_xlist_sent"),
			XListDrafts:          item.MapString("mail_xlist_drafts"),
			XListTrash:           item.MapString("mail_xlist_trash"),
			XListSpam:            item.MapString("mail_xlist_spam"),
			XListArchiv:          item.MapString("mail_xlist_archiv"),
			UsedSpace:            item.MapFloat("used_mailaccount_space"),
			IsActive:             item.MapString("mail_is_active"),
			ShowPassword:         item.MapString("show_password"),
			AllowNets:            item.MapString("mail_allow_nets"),
			TwoFA:                item.MapString("mail_2fa"),
			QuotaRule:            item.MapInt("quota_rule"),
			WebmailAutologin:     item.MapString("webmail_autologin"),
		}
	})
	if err != nil {
		return nil, err
	}
	return MailAccountList(out), nil
}

// TableHeaders returns the columns used by --output=table for
// MailAccountList.
func (MailAccountList) TableHeaders() []string {
	return []string{"LOGIN", "ADDRESS", "USED_MB", "RESPONDER", "ACTIVE", "IN_PROGRESS"}
}

// TableRows emits one row per MailAccount entry.
func (l MailAccountList) TableRows() [][]string {
	rows := make([][]string, 0, len(l))
	for _, a := range l {
		rows = append(rows, []string{
			a.Login,
			a.Addresses,
			strconv.FormatFloat(a.UsedSpace, 'f', 2, 64),
			a.Responder,
			a.IsActive,
			a.InProgress,
		})
	}
	return rows
}

// TableHeaders for the singular MailAccount view: a key/value layout
// to fit the wider field set.
func (MailAccount) TableHeaders() []string {
	return tablefmt.FieldValueHeaders
}

// TableRows emits the scalar fields. The redundant mail_adresses /
// mail_copy_adress legacy keys are omitted from the table; they remain
// available via --output=json|yaml.
func (a MailAccount) TableRows() [][]string {
	return [][]string{
		{"mail_login", a.Login},
		{"mail_addresses", a.Addresses},
		{"mail_comment", a.Comment},
		{"mail_responder", a.Responder},
		{"mail_responder_displayname", a.ResponderDisplayName},
		{"mail_responder_content_type", a.ResponderContentType},
		{"mail_copy_address", a.CopyAddress},
		{"mail_sender_alias", a.SenderAlias},
		{"mail_spamfilter", a.Spamfilter},
		{"mail_xlist_enabled", a.XListEnabled},
		{"mail_xlist_sent", a.XListSent},
		{"mail_xlist_drafts", a.XListDrafts},
		{"mail_xlist_trash", a.XListTrash},
		{"mail_xlist_spam", a.XListSpam},
		{"mail_xlist_archiv", a.XListArchiv},
		{"used_mailaccount_space", strconv.FormatFloat(a.UsedSpace, 'f', 2, 64)},
		{"mail_is_active", a.IsActive},
		{"show_password", a.ShowPassword},
		{"mail_allow_nets", a.AllowNets},
		{"mail_2fa", a.TwoFA},
		{"quota_rule", strconv.Itoa(a.QuotaRule)},
		{"webmail_autologin", a.WebmailAutologin},
		{"in_progress", a.InProgress},
	}
}
