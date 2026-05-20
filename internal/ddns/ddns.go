package ddns

import (
	"context"

	"github.com/chmmou/kasapi-cli/internal/kasread"
	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/tablefmt"
)

// Caller is the subset of *api.Client this package depends on. The
// indirection keeps tests free of network setup: a fake Caller can
// return a *soap.Response decoded from a fixture.
type Caller = kasread.Caller

// DDNSUser is one entry of get_ddnsusers. The list and singular
// views (the latter being get_ddnsusers called with a `ddns_login`
// filter, per the KAS API docs at
// https://kasapi.kasserver.com/dokumentation/phpdoc/files/get-ddnsusers-inc.html)
// return the same Map shape, so a single struct covers both.
//
// Note the wire-side asymmetry: the action name uses "ddnsusers"
// (no `y`), the filter parameter uses "ddns_login" (no `y`), but
// the response keys use the "dyndns_*" prefix (with `y`). The
// struct mirrors the wire keys verbatim.
//
// Per the captured fixtures both variants return the legacy
// `dyndns_target_ip` alongside the explicit `dyndns_target_ipv4` and
// `dyndns_target_ipv6` dual-stack fields (see
// testdata/ddns/get_ddnsuser*_response_success.xml). The ipv4/ipv6
// pair plus `in_progress` may still be absent on older accounts, so
// all three are flagged with omitempty.
type DDNSUser struct {
	Login    string `json:"dyndns_login" yaml:"dyndns_login"`
	Password string `json:"dyndns_password,omitempty" yaml:"dyndns_password,omitempty"`
	Zone     string `json:"dyndns_zone" yaml:"dyndns_zone"`
	Label    string `json:"dyndns_label" yaml:"dyndns_label"`

	TargetIP   string `json:"dyndns_target_ip" yaml:"dyndns_target_ip"`
	TargetIPv4 string `json:"dyndns_target_ipv4,omitempty" yaml:"dyndns_target_ipv4,omitempty"`
	TargetIPv6 string `json:"dyndns_target_ipv6,omitempty" yaml:"dyndns_target_ipv6,omitempty"`

	DualStack string `json:"dyndns_dual_stack" yaml:"dyndns_dual_stack"`
	Comment   string `json:"dyndns_comment" yaml:"dyndns_comment"`

	InProgress string `json:"in_progress,omitempty" yaml:"in_progress,omitempty"`
}

// FQDN returns the fully-qualified DDNS hostname (label + zone) for
// convenience in tabular output.
func (u DDNSUser) FQDN() string {
	if u.Label == "" {
		return u.Zone
	}
	if u.Zone == "" {
		return u.Label
	}
	return u.Label + "." + u.Zone
}

// DDNSUserList is the typed payload of get_ddnsusers; satisfies
// cli.Tabular.
type DDNSUserList []DDNSUser

// Client groups the read endpoints scoped to DDNS users
// (get_ddnsusers, list and singular) and the write endpoints
// add_ddnsuser / update_ddnsuser / delete_ddnsuser (see write.go).
// The raw Caller is kept alongside the read helper so the write
// methods can dispatch their own KAS actions through the shared
// kaswrite seam.
//
// The KAS API signals "filter matched no entry" with a SOAP fault
// (`dyndns_login_not_found`) rather than an empty array; that fault
// propagates through the Caller as an *api.Error and is detected by
// `api.IsNotFound`, so this client does not need a separate
// not-found path.
type Client struct {
	lg kasread.ListGet[DDNSUserList, DDNSUser]
	c  Caller
}

// NewClient returns a Client backed by the given Caller.
func NewClient(c Caller) *Client {
	return &Client{
		lg: kasread.ListGet[DDNSUserList, DDNSUser]{
			Caller:    c,
			Action:    "get_ddnsusers",
			Label:     "ddns",
			ArgName:   "login",
			FilterKey: "ddns_login",
			Decoder:   DecodeDDNSUsers,
		},
		c: c,
	}
}

// List calls get_ddnsusers without parameters and decodes the
// response into a DDNSUserList covering every DDNS user visible to
// the login.
func (c *Client) List(ctx context.Context) (DDNSUserList, error) { return c.lg.List(ctx) }

// Get calls get_ddnsusers with a ddns_login filter and returns the
// single matching DDNSUser. The KAS API still wraps the result in an
// array; we unwrap it here so callers do not have to. A guard
// against an unexpected empty array is included for parity with the
// other read modules even though the documented behaviour is a
// `dyndns_login_not_found` SOAP fault.
func (c *Client) Get(ctx context.Context, login string) (DDNSUser, error) {
	return c.lg.Get(ctx, login)
}

// DecodeDDNSUsers maps the ReturnInfo of a get_ddnsusers response
// (an Array of Maps) into the typed DDNSUserList.
func DecodeDDNSUsers(returnInfo soap.Value) (DDNSUserList, error) {
	out, err := soap.DecodeArray(returnInfo, "ddns", func(item soap.Value) DDNSUser {
		return DDNSUser{
			Login:      item.MapString("dyndns_login"),
			Password:   item.MapString("dyndns_password"),
			Zone:       item.MapString("dyndns_zone"),
			Label:      item.MapString("dyndns_label"),
			TargetIP:   item.MapString("dyndns_target_ip"),
			TargetIPv4: item.MapString("dyndns_target_ipv4"),
			TargetIPv6: item.MapString("dyndns_target_ipv6"),
			DualStack:  item.MapString("dyndns_dual_stack"),
			Comment:    item.MapString("dyndns_comment"),
			InProgress: item.MapString("in_progress"),
		}
	})
	if err != nil {
		return nil, err
	}
	return DDNSUserList(out), nil
}

// TableHeaders returns the columns used by --output=table for
// DDNSUserList.
func (DDNSUserList) TableHeaders() []string {
	return []string{"LOGIN", "FQDN", "TARGET_IP", "DUAL_STACK", "COMMENT", "IN_PROGRESS"}
}

// TableRows emits one row per DDNSUser entry. The label and zone
// are joined into a single FQDN column so the table reflects the
// hostname clients will actually look up. dyndns_password is
// intentionally omitted — consumers that need it should use
// --output=json|yaml.
func (l DDNSUserList) TableRows() [][]string {
	rows := make([][]string, 0, len(l))
	for _, u := range l {
		rows = append(rows, []string{
			u.Login,
			u.FQDN(),
			u.TargetIP,
			u.DualStack,
			u.Comment,
			u.InProgress,
		})
	}
	return rows
}

// TableHeaders for the singular DDNSUser view: a key/value layout.
func (DDNSUser) TableHeaders() []string {
	return tablefmt.FieldValueHeaders
}

// TableRows emits the scalar fields. dyndns_password is intentionally
// omitted — consumers that need it should use --output=json|yaml.
// The optional ipv4/ipv6 detail fields and in_progress only appear
// when the API actually returned them.
func (u DDNSUser) TableRows() [][]string {
	rows := [][]string{
		{"dyndns_login", u.Login},
		{"dyndns_zone", u.Zone},
		{"dyndns_label", u.Label},
		{"fqdn", u.FQDN()},
		{"dyndns_target_ip", u.TargetIP},
	}
	if u.TargetIPv4 != "" {
		rows = append(rows, []string{"dyndns_target_ipv4", u.TargetIPv4})
	}
	if u.TargetIPv6 != "" {
		rows = append(rows, []string{"dyndns_target_ipv6", u.TargetIPv6})
	}
	rows = append(rows,
		[]string{"dyndns_dual_stack", u.DualStack},
		[]string{"dyndns_comment", u.Comment},
	)
	if u.InProgress != "" {
		rows = append(rows, []string{"in_progress", u.InProgress})
	}
	return rows
}
