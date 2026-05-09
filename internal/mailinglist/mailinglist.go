package mailinglist

import (
	"context"
	"fmt"

	"github.com/chmmou/kasapi-cli/internal/soap"
)

// Caller is the subset of *api.Client this package depends on. The
// indirection keeps tests free of network setup.
type Caller interface {
	Call(ctx context.Context, action string, params map[string]any) (*soap.Response, error)
}

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
	API Caller
}

// NewClient returns a Client backed by the given Caller.
func NewClient(c Caller) *Client { return &Client{API: c} }

// List calls get_mailinglists without parameters and decodes the
// response into a MailingListList covering every mailing list visible
// to the login.
func (c *Client) List(ctx context.Context) (MailingListList, error) {
	resp, err := c.API.Call(ctx, "get_mailinglists", nil)
	if err != nil {
		return nil, err
	}
	list, err := DecodeMailingLists(resp.Body.ReturnInfo)
	if err != nil {
		return nil, fmt.Errorf("mailinglist: get_mailinglists: %w", err)
	}
	return list, nil
}

// Get calls get_mailinglists with a mailinglist_name filter and returns
// the single matching MailingList. The KAS API still wraps the result
// in an array; we unwrap it so callers do not have to. An empty array
// surfaces as a not-found error.
func (c *Client) Get(ctx context.Context, name string) (MailingList, error) {
	if name == "" {
		return MailingList{}, fmt.Errorf("mailinglist: name is required")
	}
	resp, err := c.API.Call(ctx, "get_mailinglists", map[string]any{"mailinglist_name": name})
	if err != nil {
		return MailingList{}, err
	}
	list, err := DecodeMailingLists(resp.Body.ReturnInfo)
	if err != nil {
		return MailingList{}, fmt.Errorf("mailinglist: get_mailinglists: %w", err)
	}
	if len(list) == 0 {
		return MailingList{}, fmt.Errorf("mailinglist: %q not found", name)
	}
	return list[0], nil
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
