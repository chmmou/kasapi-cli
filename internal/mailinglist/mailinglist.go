package mailinglist

import (
	"context"

	"github.com/chmmou/kasapi-cli/internal/kasread"
	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/tablefmt"
)

// Caller is the subset of *api.Client this package depends on. The
// indirection keeps tests free of network setup.
type Caller = kasread.Caller

// MailingList is one entry of get_mailinglists. The list and singular
// views (the latter being get_mailinglists called with a mailinglist_name
// filter) share this struct, but the API does NOT return the same Map
// shape for both: the list view carries only name/domain/password/
// is_active/in_progress, while the singular view additionally returns
// subscriber/config/restrict_post. The singular-only fields are kept
// omitempty so the list payload stays clean. The password follows the
// mailaccount precedent: surfaced via --output=json|yaml (omitempty),
// never in table output.
type MailingList struct {
	Name     string `json:"mailinglist_name" yaml:"mailinglist_name"`
	Domain   string `json:"mailinglist_domain" yaml:"mailinglist_domain"`
	Password string `json:"mailinglist_password,omitempty" yaml:"mailinglist_password,omitempty"`
	IsActive string `json:"mailinglist_is_active" yaml:"mailinglist_is_active"`

	// Singular-view-only (get_mailinglists with a mailinglist_name
	// filter); absent from the list view.
	Subscriber   string `json:"mailinglist_subscriber,omitempty" yaml:"mailinglist_subscriber,omitempty"`
	Config       string `json:"mailinglist_config,omitempty" yaml:"mailinglist_config,omitempty"`
	RestrictPost string `json:"mailinglist_restrict_post,omitempty" yaml:"mailinglist_restrict_post,omitempty"`

	InProgress string `json:"in_progress" yaml:"in_progress"`
}

// MailingListList is the typed payload of get_mailinglists; satisfies
// cli.Tabular.
type MailingListList []MailingList

// Client groups the read endpoints scoped to mailing lists
// (get_mailinglists, list and singular) and the write endpoints
// add_mailinglist / update_mailinglist / delete_mailinglist (see
// write.go). The raw Caller is kept alongside the read helper so the
// write methods can dispatch their own KAS actions.
type Client struct {
	lg kasread.ListGet[MailingListList, MailingList]
	c  Caller
}

// NewClient returns a Client backed by the given Caller.
func NewClient(c Caller) *Client {
	return &Client{
		lg: kasread.ListGet[MailingListList, MailingList]{
			Caller:    c,
			Action:    "get_mailinglists",
			Label:     "mailinglist",
			ArgName:   "name",
			FilterKey: "mailinglist_name",
			Decoder:   DecodeMailingLists,
		},
		c: c,
	}
}

// List calls get_mailinglists without parameters and decodes the
// response into a MailingListList covering every mailing list visible
// to the login.
func (c *Client) List(ctx context.Context) (MailingListList, error) { return c.lg.List(ctx) }

// Get calls get_mailinglists with a mailinglist_name filter and returns
// the single matching MailingList. The KAS API still wraps the result
// in an array; we unwrap it so callers do not have to. An empty array
// surfaces as a not-found error.
func (c *Client) Get(ctx context.Context, name string) (MailingList, error) {
	return c.lg.Get(ctx, name)
}

// DecodeMailingLists maps the ReturnInfo of a get_mailinglists response
// (an Array of Maps) into the typed MailingListList.
func DecodeMailingLists(returnInfo soap.Value) (MailingListList, error) {
	out, err := soap.DecodeArray(returnInfo, "mailinglist", func(item soap.Value) MailingList {
		return MailingList{
			Name:         item.MapString("mailinglist_name"),
			Domain:       item.MapString("mailinglist_domain"),
			Password:     item.MapString("mailinglist_password"),
			IsActive:     item.MapString("mailinglist_is_active"),
			Subscriber:   item.MapString("mailinglist_subscriber"),
			Config:       item.MapString("mailinglist_config"),
			RestrictPost: item.MapString("mailinglist_restrict_post"),
			InProgress:   item.MapString("in_progress"),
		}
	})
	if err != nil {
		return nil, err
	}
	return MailingListList(out), nil
}

// TableHeaders returns the columns used by --output=table for
// MailingListList.
func (MailingListList) TableHeaders() []string {
	return []string{"NAME", "DOMAIN", "ACTIVE", "IN_PROGRESS"}
}

// TableRows emits one row per MailingList entry. The password is never
// rendered in table output (mailaccount precedent); the singular-only
// subscriber/config/restrict_post fields are absent from the list view.
func (l MailingListList) TableRows() [][]string {
	rows := make([][]string, 0, len(l))
	for _, m := range l {
		rows = append(rows, []string{
			m.Name,
			m.Domain,
			m.IsActive,
			m.InProgress,
		})
	}
	return rows
}

// TableHeaders for the singular MailingList view: a key/value layout
// keeps the URL readable without truncation.
func (MailingList) TableHeaders() []string {
	return tablefmt.FieldValueHeaders
}

// TableRows emits the scalar fields of a single MailingList. The
// password is intentionally omitted from table output (mailaccount
// precedent); it remains available via --output=json|yaml.
func (m MailingList) TableRows() [][]string {
	return [][]string{
		{"mailinglist_name", m.Name},
		{"mailinglist_domain", m.Domain},
		{"mailinglist_is_active", m.IsActive},
		{"mailinglist_subscriber", m.Subscriber},
		{"mailinglist_config", m.Config},
		{"mailinglist_restrict_post", m.RestrictPost},
		{"in_progress", m.InProgress},
	}
}
