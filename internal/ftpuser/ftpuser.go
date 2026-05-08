package ftpuser

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

// FTPUser is one entry of get_ftpusers. List and singular views (the
// latter being get_ftpusers called with an ftp_login filter) return
// the same Map shape, so a single struct covers both.
//
// The KAS API returns both the canonical ftp_password and the legacy
// German ftp_passwort key with identical content; both are kept in
// the struct so a JSON/YAML round-trip preserves the raw payload.
// Consumers should prefer Password.
type FTPUser struct {
	Login    string `json:"ftp_login" yaml:"ftp_login"`
	Password string `json:"ftp_password,omitempty" yaml:"ftp_password,omitempty"`
	Passwort string `json:"ftp_passwort,omitempty" yaml:"ftp_passwort,omitempty"`

	Path       string `json:"ftp_path" yaml:"ftp_path"`
	Comment    string `json:"ftp_comment" yaml:"ftp_comment"`
	IsMainUser string `json:"ftp_is_main_user" yaml:"ftp_is_main_user"`

	PermissionList  string `json:"ftp_permission_list" yaml:"ftp_permission_list"`
	PermissionRead  string `json:"ftp_permission_read" yaml:"ftp_permission_read"`
	PermissionWrite string `json:"ftp_permission_write" yaml:"ftp_permission_write"`

	VirusClamAV string `json:"ftp_virus_clamav" yaml:"ftp_virus_clamav"`
	InProgress  string `json:"in_progress" yaml:"in_progress"`
}

// FTPUserList is the typed payload of get_ftpusers; satisfies
// cli.Tabular.
type FTPUserList []FTPUser

// Client groups the read endpoints scoped to FTP users:
// get_ftpusers (list and singular).
type Client struct {
	API Caller
}

// NewClient returns a Client backed by the given Caller.
func NewClient(c Caller) *Client { return &Client{API: c} }

// List calls get_ftpusers without parameters and decodes the response
// into an FTPUserList covering every FTP user visible to the login.
func (c *Client) List(ctx context.Context) (FTPUserList, error) {
	resp, err := c.API.Call(ctx, "get_ftpusers", nil)
	if err != nil {
		return nil, err
	}
	list, err := DecodeFTPUsers(resp.Body.ReturnInfo)
	if err != nil {
		return nil, fmt.Errorf("ftpuser: get_ftpusers: %w", err)
	}
	return list, nil
}

// Get calls get_ftpusers with an ftp_login filter and returns the
// single matching FTPUser. The KAS API still wraps the result in an
// array; we unwrap it here so callers do not have to. An empty array
// surfaces as a not-found error.
func (c *Client) Get(ctx context.Context, login string) (FTPUser, error) {
	if login == "" {
		return FTPUser{}, fmt.Errorf("ftpuser: login is required")
	}
	resp, err := c.API.Call(ctx, "get_ftpusers", map[string]any{"ftp_login": login})
	if err != nil {
		return FTPUser{}, err
	}
	list, err := DecodeFTPUsers(resp.Body.ReturnInfo)
	if err != nil {
		return FTPUser{}, fmt.Errorf("ftpuser: get_ftpusers: %w", err)
	}
	if len(list) == 0 {
		return FTPUser{}, fmt.Errorf("ftpuser: %q not found", login)
	}
	return list[0], nil
}

// DecodeFTPUsers maps the ReturnInfo of a get_ftpusers response (an
// Array of Maps) into the typed FTPUserList.
func DecodeFTPUsers(returnInfo soap.Value) (FTPUserList, error) {
	if returnInfo.Kind != soap.KindArray {
		return nil, fmt.Errorf("ftpuser: expected ReturnInfo array, got kind %d", returnInfo.Kind)
	}
	out := make(FTPUserList, 0, len(returnInfo.Array))
	for i, item := range returnInfo.Array {
		if item.Kind != soap.KindMap {
			return nil, fmt.Errorf("ftpuser: ReturnInfo[%d] is not a Map", i)
		}
		out = append(out, FTPUser{
			Login:           item.MapString("ftp_login"),
			Password:        item.MapString("ftp_password"),
			Passwort:        item.MapString("ftp_passwort"),
			Path:            item.MapString("ftp_path"),
			Comment:         item.MapString("ftp_comment"),
			IsMainUser:      item.MapString("ftp_is_main_user"),
			PermissionList:  item.MapString("ftp_permission_list"),
			PermissionRead:  item.MapString("ftp_permission_read"),
			PermissionWrite: item.MapString("ftp_permission_write"),
			VirusClamAV:     item.MapString("ftp_virus_clamav"),
			InProgress:      item.MapString("in_progress"),
		})
	}
	return out, nil
}

// TableHeaders returns the columns used by --output=table for
// FTPUserList.
func (FTPUserList) TableHeaders() []string {
	return []string{"LOGIN", "PATH", "COMMENT", "MAIN", "R", "W", "L", "CLAMAV", "IN_PROGRESS"}
}

// TableRows emits one row per FTPUser entry. The R/W/L columns mirror
// the KAS UI layout for the three permission flags
// (read/write/list-only) so a quick scan shows what each account can do.
func (l FTPUserList) TableRows() [][]string {
	rows := make([][]string, 0, len(l))
	for _, u := range l {
		rows = append(rows, []string{
			u.Login,
			u.Path,
			u.Comment,
			u.IsMainUser,
			u.PermissionRead,
			u.PermissionWrite,
			u.PermissionList,
			u.VirusClamAV,
			u.InProgress,
		})
	}
	return rows
}

// TableHeaders for the singular FTPUser view: a key/value layout.
func (FTPUser) TableHeaders() []string {
	return []string{"FIELD", "VALUE"}
}

// TableRows emits the scalar fields. ftp_password / ftp_passwort are
// intentionally omitted — consumers that need them should use
// --output=json|yaml.
func (u FTPUser) TableRows() [][]string {
	return [][]string{
		{"ftp_login", u.Login},
		{"ftp_path", u.Path},
		{"ftp_comment", u.Comment},
		{"ftp_is_main_user", u.IsMainUser},
		{"ftp_permission_read", u.PermissionRead},
		{"ftp_permission_write", u.PermissionWrite},
		{"ftp_permission_list", u.PermissionList},
		{"ftp_virus_clamav", u.VirusClamAV},
		{"in_progress", u.InProgress},
	}
}
