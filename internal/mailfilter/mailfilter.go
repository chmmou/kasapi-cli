package mailfilter

import (
	"context"

	"github.com/chmmou/kasapi-cli/internal/kasread"
	"github.com/chmmou/kasapi-cli/internal/soap"
)

// Caller is the subset of *api.Client this package depends on. The
// indirection keeps tests free of network setup.
type Caller = kasread.Caller

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
// Get is intentionally absent — the KAS endpoint does not document a
// filter parameter (see issue #73 NTH bundle), so only List is wired
// up here.
type Client struct {
	lg kasread.ListGet[StandardFilterList, StandardFilter]
}

// NewClient returns a Client backed by the given Caller.
func NewClient(c Caller) *Client {
	return &Client{lg: kasread.ListGet[StandardFilterList, StandardFilter]{
		Caller:  c,
		Action:  "get_mailstandardfilter",
		Label:   "mailfilter",
		Decoder: DecodeStandardFilters,
	}}
}

// List calls get_mailstandardfilter and decodes the response into a
// StandardFilterList. The endpoint takes no parameters.
func (c *Client) List(ctx context.Context) (StandardFilterList, error) { return c.lg.List(ctx) }

// DecodeStandardFilters maps the ReturnInfo of a get_mailstandardfilter
// response (an Array of Maps) into the typed StandardFilterList.
func DecodeStandardFilters(returnInfo soap.Value) (StandardFilterList, error) {
	out, err := soap.DecodeArray(returnInfo, "mailfilter", func(item soap.Value) StandardFilter {
		return StandardFilter{
			Filter:      item.MapString("filter"),
			Type:        item.MapString("type"),
			Title:       item.MapString("title"),
			Recommended: item.MapString("recommended"),
		}
	})
	if err != nil {
		return nil, err
	}
	return StandardFilterList(out), nil
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
