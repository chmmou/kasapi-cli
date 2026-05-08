package mailfilter

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

// StandardFilter is one entry of get_mailstandardfilter, describing a
// preset spam/virus filter that can be referenced by `mail_spamfilter`
// on a mail account or forward.
type StandardFilter struct {
	Filter      string `json:"filter" yaml:"filter"`
	Type        string `json:"type" yaml:"type"`
	Title       string `json:"title" yaml:"title"`
	Recommended string `json:"recommended" yaml:"recommended"`
}

// StandardFilterList is the typed payload of get_mailstandardfilter;
// satisfies cli.Tabular.
type StandardFilterList []StandardFilter

// Client groups the read endpoint scoped to mail standard filters.
type Client struct {
	API Caller
}

// NewClient returns a Client backed by the given Caller.
func NewClient(c Caller) *Client { return &Client{API: c} }

// List calls get_mailstandardfilter and decodes the response into a
// StandardFilterList. The endpoint takes no parameters.
func (c *Client) List(ctx context.Context) (StandardFilterList, error) {
	resp, err := c.API.Call(ctx, "get_mailstandardfilter", nil)
	if err != nil {
		return nil, err
	}
	list, err := DecodeStandardFilters(resp.Body.ReturnInfo)
	if err != nil {
		return nil, fmt.Errorf("mailfilter: get_mailstandardfilter: %w", err)
	}
	return list, nil
}

// DecodeStandardFilters maps the ReturnInfo of a get_mailstandardfilter
// response (an Array of Maps) into the typed StandardFilterList.
func DecodeStandardFilters(returnInfo soap.Value) (StandardFilterList, error) {
	if returnInfo.Kind != soap.KindArray {
		return nil, fmt.Errorf("mailfilter: expected ReturnInfo array, got kind %d", returnInfo.Kind)
	}
	out := make(StandardFilterList, 0, len(returnInfo.Array))
	for i, item := range returnInfo.Array {
		if item.Kind != soap.KindMap {
			return nil, fmt.Errorf("mailfilter: ReturnInfo[%d] is not a Map", i)
		}
		out = append(out, StandardFilter{
			Filter:      item.MapString("filter"),
			Type:        item.MapString("type"),
			Title:       item.MapString("title"),
			Recommended: item.MapString("recommended"),
		})
	}
	return out, nil
}

// TableHeaders returns the columns used by --output=table for
// StandardFilterList.
func (StandardFilterList) TableHeaders() []string {
	return []string{"FILTER", "TYPE", "TITLE", "RECOMMENDED"}
}

// TableRows emits one row per StandardFilter entry.
func (l StandardFilterList) TableRows() [][]string {
	rows := make([][]string, 0, len(l))
	for _, f := range l {
		rows = append(rows, []string{
			f.Filter,
			f.Type,
			f.Title,
			f.Recommended,
		})
	}
	return rows
}
