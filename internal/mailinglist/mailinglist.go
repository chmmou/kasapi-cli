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

// MailingList is one entry of get_mailinglists, describing a Mailman
// list provisioned for the account.
type MailingList struct {
	Name       string `json:"mailinglist_name" yaml:"mailinglist_name"`
	Admin      string `json:"mailinglist_admin" yaml:"mailinglist_admin"`
	URL        string `json:"mailinglist_url" yaml:"mailinglist_url"`
	InProgress string `json:"in_progress" yaml:"in_progress"`
}

// MailingListList is the typed payload of get_mailinglists; satisfies
// cli.Tabular.
type MailingListList []MailingList

// Client groups the read endpoint scoped to mailing lists.
type Client struct {
	API Caller
}

// NewClient returns a Client backed by the given Caller.
func NewClient(c Caller) *Client { return &Client{API: c} }

// List calls get_mailinglists and decodes the response into a
// MailingListList covering every mailing list visible to the login.
// The endpoint takes no parameters.
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

// DecodeMailingLists maps the ReturnInfo of a get_mailinglists response
// (an Array of Maps) into the typed MailingListList.
func DecodeMailingLists(returnInfo soap.Value) (MailingListList, error) {
	if returnInfo.Kind != soap.KindArray {
		return nil, fmt.Errorf("mailinglist: expected ReturnInfo array, got kind %d", returnInfo.Kind)
	}
	out := make(MailingListList, 0, len(returnInfo.Array))
	for i, item := range returnInfo.Array {
		if item.Kind != soap.KindMap {
			return nil, fmt.Errorf("mailinglist: ReturnInfo[%d] is not a Map", i)
		}
		out = append(out, MailingList{
			Name:       item.MapString("mailinglist_name"),
			Admin:      item.MapString("mailinglist_admin"),
			URL:        item.MapString("mailinglist_url"),
			InProgress: item.MapString("in_progress"),
		})
	}
	return out, nil
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
