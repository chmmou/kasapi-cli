package mailaccount

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/chmmou/kasapi-cli/internal/soap"
)

// Caller is the subset of *api.Client this package depends on. The
// indirection keeps tests free of network setup: a fake Caller can
// return a *soap.Response decoded from a fixture.
type Caller interface {
	Call(ctx context.Context, action string, params map[string]any) (*soap.Response, error)
}

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
	API Caller
}

// NewClient returns a Client backed by the given Caller.
func NewClient(c Caller) *Client { return &Client{API: c} }

// List calls get_mailaccounts without parameters and decodes the
// response into a MailAccountList covering every mail account visible
// to the login.
func (c *Client) List(ctx context.Context) (MailAccountList, error) {
	resp, err := c.API.Call(ctx, "get_mailaccounts", nil)
	if err != nil {
		return nil, err
	}
	list, err := DecodeMailAccounts(resp.Body.ReturnInfo)
	if err != nil {
		return nil, fmt.Errorf("mailaccount: get_mailaccounts: %w", err)
	}
	return list, nil
}

// Get calls get_mailaccounts with a mail_login filter and returns the
// single matching MailAccount. The KAS API still wraps the result in
// an array; we unwrap it here so callers do not have to. An empty
// array surfaces as a not-found error.
func (c *Client) Get(ctx context.Context, login string) (MailAccount, error) {
	if login == "" {
		return MailAccount{}, fmt.Errorf("mailaccount: login is required")
	}
	resp, err := c.API.Call(ctx, "get_mailaccounts", map[string]any{"mail_login": login})
	if err != nil {
		return MailAccount{}, err
	}
	list, err := DecodeMailAccounts(resp.Body.ReturnInfo)
	if err != nil {
		return MailAccount{}, fmt.Errorf("mailaccount: get_mailaccounts: %w", err)
	}
	if len(list) == 0 {
		return MailAccount{}, fmt.Errorf("mailaccount: %q not found", login)
	}
	return list[0], nil
}

// DecodeMailAccounts maps the ReturnInfo of a get_mailaccounts response
// (an Array of Maps) into the typed MailAccountList.
func DecodeMailAccounts(returnInfo soap.Value) (MailAccountList, error) {
	if returnInfo.Kind != soap.KindArray {
		return nil, fmt.Errorf("mailaccount: expected ReturnInfo array, got kind %d", returnInfo.Kind)
	}
	out := make(MailAccountList, 0, len(returnInfo.Array))
	for i, item := range returnInfo.Array {
		if item.Kind != soap.KindMap {
			return nil, fmt.Errorf("mailaccount: ReturnInfo[%d] is not a Map", i)
		}
		out = append(out, MailAccount{
			Login:                getString(item, "mail_login"),
			Password:             getString(item, "mail_password"),
			Adresses:             getString(item, "mail_adresses"),
			Addresses:            getString(item, "mail_addresses"),
			Comment:              getString(item, "mail_comment"),
			Responder:            getString(item, "mail_responder"),
			ResponderText:        getString(item, "mail_responder_text"),
			ResponderDisplayName: getString(item, "mail_responder_displayname"),
			ResponderContentType: getString(item, "mail_responder_content_type"),
			CopyAdress:           getString(item, "mail_copy_adress"),
			CopyAddress:          getString(item, "mail_copy_address"),
			SenderAlias:          getString(item, "mail_sender_alias"),
			Spamfilter:           getString(item, "mail_spamfilter"),
			InProgress:           getString(item, "in_progress"),
			XListEnabled:         getString(item, "mail_xlist_enabled"),
			XListSent:            getString(item, "mail_xlist_sent"),
			XListDrafts:          getString(item, "mail_xlist_drafts"),
			XListTrash:           getString(item, "mail_xlist_trash"),
			XListSpam:            getString(item, "mail_xlist_spam"),
			XListArchiv:          getString(item, "mail_xlist_archiv"),
			UsedSpace:            getFloat(item, "used_mailaccount_space"),
			IsActive:             getString(item, "mail_is_active"),
			ShowPassword:         getString(item, "show_password"),
			AllowNets:            getString(item, "mail_allow_nets"),
			TwoFA:                getString(item, "mail_2fa"),
			QuotaRule:            getInt(item, "quota_rule"),
			WebmailAutologin:     getString(item, "webmail_autologin"),
		})
	}
	return out, nil
}

func getString(m soap.Value, key string) string {
	v, ok := m.Get(key)
	if !ok {
		return ""
	}
	return v.AsString()
}

func getInt(m soap.Value, key string) int {
	v, ok := m.Get(key)
	if !ok {
		return 0
	}
	switch v.Kind {
	case soap.KindInt:
		return int(v.Int)
	case soap.KindFloat:
		return int(v.Float)
	case soap.KindString:
		s := strings.TrimSpace(v.String)
		if s == "" {
			return 0
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return 0
		}
		return n
	}
	return 0
}

func getFloat(m soap.Value, key string) float64 {
	v, ok := m.Get(key)
	if !ok {
		return 0
	}
	return v.AsFloat()
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
	return []string{"FIELD", "VALUE"}
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
