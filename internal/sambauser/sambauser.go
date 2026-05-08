package sambauser

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

// SambaUser is one entry of get_sambausers. The list and singular
// views (the latter being get_sambausers called with a samba_login
// filter, per the KAS API docs at
// https://kasapi.kasserver.com/dokumentation/phpdoc/files/get-sambausers-inc.html)
// return the same Map shape, so a single struct covers both.
type SambaUser struct {
	Login      string `json:"samba_login" yaml:"samba_login"`
	Password   string `json:"samba_password,omitempty" yaml:"samba_password,omitempty"`
	Path       string `json:"samba_path" yaml:"samba_path"`
	Comment    string `json:"samba_comment" yaml:"samba_comment"`
	InProgress string `json:"in_progress" yaml:"in_progress"`
}

// SambaUserList is the typed payload of get_sambausers; satisfies
// cli.Tabular.
type SambaUserList []SambaUser

// Client groups the read endpoints scoped to Samba users:
// get_sambausers (list and singular).
type Client struct {
	API Caller
}

// NewClient returns a Client backed by the given Caller.
func NewClient(c Caller) *Client { return &Client{API: c} }

// List calls get_sambausers without parameters and decodes the
// response into a SambaUserList covering every Samba user visible to
// the login.
func (c *Client) List(ctx context.Context) (SambaUserList, error) {
	resp, err := c.API.Call(ctx, "get_sambausers", nil)
	if err != nil {
		return nil, err
	}
	list, err := DecodeSambaUsers(resp.Body.ReturnInfo)
	if err != nil {
		return nil, fmt.Errorf("sambauser: get_sambausers: %w", err)
	}
	return list, nil
}

// Get calls get_sambausers with a samba_login filter and returns the
// single matching SambaUser. The KAS API still wraps the result in
// an array; we unwrap it here so callers do not have to. An empty
// array surfaces as a not-found error.
func (c *Client) Get(ctx context.Context, login string) (SambaUser, error) {
	if login == "" {
		return SambaUser{}, fmt.Errorf("sambauser: login is required")
	}
	resp, err := c.API.Call(ctx, "get_sambausers", map[string]any{"samba_login": login})
	if err != nil {
		return SambaUser{}, err
	}
	list, err := DecodeSambaUsers(resp.Body.ReturnInfo)
	if err != nil {
		return SambaUser{}, fmt.Errorf("sambauser: get_sambausers: %w", err)
	}
	if len(list) == 0 {
		return SambaUser{}, fmt.Errorf("sambauser: %q not found", login)
	}
	return list[0], nil
}

// DecodeSambaUsers maps the ReturnInfo of a get_sambausers response
// (an Array of Maps) into the typed SambaUserList.
func DecodeSambaUsers(returnInfo soap.Value) (SambaUserList, error) {
	if returnInfo.Kind != soap.KindArray {
		return nil, fmt.Errorf("sambauser: expected ReturnInfo array, got kind %d", returnInfo.Kind)
	}
	out := make(SambaUserList, 0, len(returnInfo.Array))
	for i, item := range returnInfo.Array {
		if item.Kind != soap.KindMap {
			return nil, fmt.Errorf("sambauser: ReturnInfo[%d] is not a Map", i)
		}
		out = append(out, SambaUser{
			Login:      item.MapString("samba_login"),
			Password:   item.MapString("samba_password"),
			Path:       item.MapString("samba_path"),
			Comment:    item.MapString("samba_comment"),
			InProgress: item.MapString("in_progress"),
		})
	}
	return out, nil
}

// TableHeaders returns the columns used by --output=table for
// SambaUserList.
func (SambaUserList) TableHeaders() []string {
	return []string{"LOGIN", "PATH", "COMMENT", "IN_PROGRESS"}
}

// TableRows emits one row per SambaUser entry. samba_password is
// intentionally omitted — consumers that need it should use
// --output=json|yaml.
func (l SambaUserList) TableRows() [][]string {
	rows := make([][]string, 0, len(l))
	for _, u := range l {
		rows = append(rows, []string{
			u.Login,
			u.Path,
			u.Comment,
			u.InProgress,
		})
	}
	return rows
}

// TableHeaders for the singular SambaUser view: a key/value layout
// matches the rest of the singular detail commands.
func (SambaUser) TableHeaders() []string {
	return []string{"FIELD", "VALUE"}
}

// TableRows emits the scalar fields. samba_password is intentionally
// omitted — consumers that need it should use --output=json|yaml.
func (u SambaUser) TableRows() [][]string {
	return [][]string{
		{"samba_login", u.Login},
		{"samba_path", u.Path},
		{"samba_comment", u.Comment},
		{"in_progress", u.InProgress},
	}
}
