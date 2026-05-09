package mailinglist

import (
	"context"

	"github.com/chmmou/kasapi-cli/internal/kasread"
	"github.com/chmmou/kasapi-cli/internal/soap"
)

// Caller is the subset of *api.Client this package depends on. The
// indirection keeps tests free of network setup.
type Caller = kasread.Caller

// MailingList is one entry of get_mailinglists. The list and singular
// views (the latter being get_mailinglists called with a mailinglist_name
// filter) return the same Map shape, so a single struct covers both.
type MailingList struct {
	Name       string `json:"mailinglist_name" yaml:"mailinglist_name"`
	Admin      string `json:"mailinglist_admin" yaml:"mailinglist_admin"`
	URL        string `json:"mailinglist_url" yaml:"mailinglist_url"`
	InProgress string `json:"in_progress" yaml:"in_progress"`
}

// MailingListList is the typed payload of get_mailinglists; satisfies
// cli.Tabular.
type MailingListList []MailingList

// Client groups the read endpoints scoped to mailing lists:
// get_mailinglists (list and singular).
type Client struct {
	lg kasread.ListGet[MailingListList, MailingList]
}

// NewClient returns a Client backed by the given Caller.
func NewClient(c Caller) *Client {
	return &Client{lg: kasread.ListGet[MailingListList, MailingList]{
		Caller:    c,
		Action:    "get_mailinglists",
		Label:     "mailinglist",
		ArgName:   "name",
		FilterKey: "mailinglist_name",
		Decoder:   DecodeMailingLists,
	}}
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
			Name:       item.MapString("mailinglist_name"),
			Admin:      item.MapString("mailinglist_admin"),
			URL:        item.MapString("mailinglist_url"),
			InProgress: item.MapString("in_progress"),
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
	return []string{"NAME", "ADMIN", "URL", "IN_PROGRESS"}
}

// TableRows emits one row per MailingList entry.
func (l MailingListList) TableRows() [][]string {
	rows := make([][]string, 0, len(l))
	for _, m := range l {
		rows = append(rows, []string{
			m.Name,
			m.Admin,
			m.URL,
			m.InProgress,
		})
	}
	return rows
}

// TableHeaders for the singular MailingList view: a key/value layout
// keeps the URL readable without truncation.
func (MailingList) TableHeaders() []string {
	return []string{"FIELD", "VALUE"}
}

// TableRows emits the scalar fields of a single MailingList.
func (m MailingList) TableRows() [][]string {
	return [][]string{
		{"mailinglist_name", m.Name},
		{"mailinglist_admin", m.Admin},
		{"mailinglist_url", m.URL},
		{"in_progress", m.InProgress},
	}
}
