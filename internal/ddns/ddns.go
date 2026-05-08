package ddns

import (
	"context"
	"fmt"

	"github.com/chmmou/kasapi-cli/internal/soap"
)

// Caller is the subset of *api.Client this package depends on. The
// indirection keeps tests free of network setup: a fake Caller can
// return a *soap.Response decoded from a fixture.
type Caller interface {
	Call(ctx context.Context, action string, params map[string]any) (*soap.Response, error)
}

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
// The list endpoint returns only the legacy `dyndns_target_ip`;
// the singular variant additionally returns `dyndns_target_ipv4`
// and `dyndns_target_ipv6` for explicit dual-stack inspection.
// Those plus `in_progress` may be absent on either variant, so all
// three are flagged with omitempty.
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

// Client groups the read endpoints scoped to DDNS users:
// get_ddnsusers (list and singular).
//
// The KAS API signals "filter matched no entry" with a SOAP fault
// (`dyndns_login_not_found`) rather than an empty array; that fault
// propagates through the Caller as an *api.Error and is detected by
// `api.IsNotFound`, so this client does not need a separate
// not-found path.
type Client struct {
	API Caller
}

// NewClient returns a Client backed by the given Caller.
func NewClient(c Caller) *Client { return &Client{API: c} }

// List calls get_ddnsusers without parameters and decodes the
// response into a DDNSUserList covering every DDNS user visible to
// the login.
func (c *Client) List(ctx context.Context) (DDNSUserList, error) {
	resp, err := c.API.Call(ctx, "get_ddnsusers", nil)
	if err != nil {
		return nil, err
	}
	list, err := DecodeDDNSUsers(resp.Body.ReturnInfo)
	if err != nil {
		return nil, fmt.Errorf("ddns: get_ddnsusers: %w", err)
	}
	return list, nil
}

// Get calls get_ddnsusers with a dyndns_login filter and returns the
// single matching DDNSUser. The KAS API still wraps the result in an
// array; we unwrap it here so callers do not have to. A guard
// against an unexpected empty array is included for parity with the
// other read modules even though the documented behaviour is a
// `dyndns_login_not_found` SOAP fault.
func (c *Client) Get(ctx context.Context, login string) (DDNSUser, error) {
	if login == "" {
		return DDNSUser{}, fmt.Errorf("ddns: login is required")
	}
	resp, err := c.API.Call(ctx, "get_ddnsusers", map[string]any{"ddns_login": login})
	if err != nil {
		return DDNSUser{}, err
	}
	list, err := DecodeDDNSUsers(resp.Body.ReturnInfo)
	if err != nil {
		return DDNSUser{}, fmt.Errorf("ddns: get_ddnsusers: %w", err)
	}
	if len(list) == 0 {
		return DDNSUser{}, fmt.Errorf("ddns: %q not found", login)
	}
	return list[0], nil
}

// DecodeDDNSUsers maps the ReturnInfo of a get_ddnsusers response
// (an Array of Maps) into the typed DDNSUserList.
func DecodeDDNSUsers(returnInfo soap.Value) (DDNSUserList, error) {
	if returnInfo.Kind != soap.KindArray {
		return nil, fmt.Errorf("ddns: expected ReturnInfo array, got kind %d", returnInfo.Kind)
	}
	out := make(DDNSUserList, 0, len(returnInfo.Array))
	for i, item := range returnInfo.Array {
		if item.Kind != soap.KindMap {
			return nil, fmt.Errorf("ddns: ReturnInfo[%d] is not a Map", i)
		}
		out = append(out, DDNSUser{
			Login:      getString(item, "dyndns_login"),
			Password:   getString(item, "dyndns_password"),
			Zone:       getString(item, "dyndns_zone"),
			Label:      getString(item, "dyndns_label"),
			TargetIP:   getString(item, "dyndns_target_ip"),
			TargetIPv4: getString(item, "dyndns_target_ipv4"),
			TargetIPv6: getString(item, "dyndns_target_ipv6"),
			DualStack:  getString(item, "dyndns_dual_stack"),
			Comment:    getString(item, "dyndns_comment"),
			InProgress: getString(item, "in_progress"),
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
	return []string{"FIELD", "VALUE"}
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
