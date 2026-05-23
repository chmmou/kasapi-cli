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

// Client groups the read endpoint scoped to mail standard filters
// (get_mailstandardfilter, which lists the filter catalogue available to
// the contract) and the write endpoints add_mailstandardfilter /
// delete_mailstandardfilter (see write.go). The raw Caller is kept
// alongside the read helper so the write methods can dispatch their own
// KAS actions through the shared kaswrite seam.
//
// Get is intentionally absent on the read side: the KAS docs at
// https://kasapi.kasserver.com/dokumentation/phpdoc/files/get-mailstandardfilter-inc.html
// list only the three standard auth parameters (kas_login,
// kas_auth_data, kas_auth_type) — there is no filter parameter for
// looking up a single filter, the endpoint always returns the full
// catalogue. Verified against the KAS docs while closing the
// mailfilter.Get follow-up from issue #73. The chain *configured* on a
// given mailbox is reported by get_mailaccounts in the
// mail_spamfilter field; callers wanting to confirm the outcome of an
// add/delete on a specific account read it from there.
type Client struct {
	lg kasread.ListGet[StandardFilterList, StandardFilter]
	c  Caller
}

// NewClient returns a Client backed by the given Caller.
func NewClient(c Caller) *Client {
	return &Client{
		lg: kasread.ListGet[StandardFilterList, StandardFilter]{
			Caller:  c,
			Action:  "get_mailstandardfilter",
			Label:   "mailfilter",
			Decoder: DecodeStandardFilters,
		},
		c: c,
	}
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
