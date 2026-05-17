package mailforward

import (
	"context"

	"github.com/chmmou/kasapi-cli/internal/kasread"
	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/tablefmt"
)

// Caller is the subset of *api.Client this package depends on. The
// indirection keeps tests free of network setup.
type Caller = kasread.Caller

// MailForward is one entry of get_mailforwards. The list and singular
// views (the latter being get_mailforwards called with a mail_forward
// filter) return the same Map shape, so a single struct covers both.
type MailForward struct {
	// The KAS API returns both the legacy mail_forward_adress key
	// (single d) and the canonical mail_forward_address key with
	// identical content. We keep both so a JSON/YAML round-trip
	// preserves the raw payload, but consumers should prefer Address.
	Adress  string `json:"mail_forward_adress" yaml:"mail_forward_adress"`
	Address string `json:"mail_forward_address" yaml:"mail_forward_address"`

	Comment    string `json:"mail_forward_comment" yaml:"mail_forward_comment"`
	Targets    string `json:"mail_forward_targets" yaml:"mail_forward_targets"`
	Spamfilter string `json:"mail_forward_spamfilter" yaml:"mail_forward_spamfilter"`
	InProgress string `json:"in_progress" yaml:"in_progress"`
}

// MailForwardList is the typed payload of get_mailforwards; satisfies
// cli.Tabular.
type MailForwardList []MailForward

// Client groups the read endpoints scoped to mail forwards
// (get_mailforwards, list and singular) and the write endpoints
// add_mailforward / update_mailforward / delete_mailforward (see
// write.go). The raw Caller is kept alongside the read helper so the
// write methods can dispatch their own KAS actions.
type Client struct {
	lg kasread.ListGet[MailForwardList, MailForward]
	c  Caller
}

// NewClient returns a Client backed by the given Caller.
func NewClient(c Caller) *Client {
	return &Client{
		lg: kasread.ListGet[MailForwardList, MailForward]{
			Caller:    c,
			Action:    "get_mailforwards",
			Label:     "mailforward",
			ArgName:   "address",
			FilterKey: "mail_forward",
			Decoder:   DecodeMailForwards,
		},
		c: c,
	}
}

// List calls get_mailforwards without parameters and decodes the
// response into a MailForwardList covering every mail forward visible
// to the login.
func (c *Client) List(ctx context.Context) (MailForwardList, error) { return c.lg.List(ctx) }

// Get calls get_mailforwards with a mail_forward filter (the source
// address) and returns the single matching MailForward. The KAS API
// still wraps the result in an array; we unwrap it so callers do not
// have to. An empty array surfaces as a not-found error.
func (c *Client) Get(ctx context.Context, address string) (MailForward, error) {
	return c.lg.Get(ctx, address)
}

// DecodeMailForwards maps the ReturnInfo of a get_mailforwards response
// (an Array of Maps) into the typed MailForwardList.
func DecodeMailForwards(returnInfo soap.Value) (MailForwardList, error) {
	out, err := soap.DecodeArray(returnInfo, "mailforward", func(item soap.Value) MailForward {
		return MailForward{
			Adress:     item.MapString("mail_forward_adress"),
			Address:    item.MapString("mail_forward_address"),
			Comment:    item.MapString("mail_forward_comment"),
			Targets:    item.MapString("mail_forward_targets"),
			Spamfilter: item.MapString("mail_forward_spamfilter"),
			InProgress: item.MapString("in_progress"),
		}
	})
	if err != nil {
		return nil, err
	}
	return MailForwardList(out), nil
}

// TableHeaders returns the columns used by --output=table for
// MailForwardList.
func (MailForwardList) TableHeaders() []string {
	return []string{"ADDRESS", "TARGETS", "SPAMFILTER", "COMMENT", "IN_PROGRESS"}
}

// TableRows emits one row per MailForward entry.
func (l MailForwardList) TableRows() [][]string {
	rows := make([][]string, 0, len(l))
	for _, f := range l {
		rows = append(rows, []string{
			f.Address,
			f.Targets,
			f.Spamfilter,
			f.Comment,
			f.InProgress,
		})
	}
	return rows
}

// TableHeaders for the singular MailForward view: a key/value layout
// to keep multi-target lists readable.
func (MailForward) TableHeaders() []string {
	return tablefmt.FieldValueHeaders
}

// TableRows emits the scalar fields. The redundant mail_forward_adress
// legacy key is omitted from the table; it remains available via
// --output=json|yaml.
func (f MailForward) TableRows() [][]string {
	return [][]string{
		{"mail_forward_address", f.Address},
		{"mail_forward_targets", f.Targets},
		{"mail_forward_spamfilter", f.Spamfilter},
		{"mail_forward_comment", f.Comment},
		{"in_progress", f.InProgress},
	}
}
