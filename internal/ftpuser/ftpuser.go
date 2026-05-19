package ftpuser

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

// Client groups the read endpoint scoped to FTP users (get_ftpusers,
// list and singular) and the write endpoints add_ftpuser /
// update_ftpuser / delete_ftpuser (see write.go). The raw Caller is
// kept alongside the read helper so the write methods can dispatch
// their own KAS actions through the shared kaswrite seam.
type Client struct {
	lg kasread.ListGet[FTPUserList, FTPUser]
	c  Caller
}

// NewClient returns a Client backed by the given Caller.
func NewClient(c Caller) *Client {
	return &Client{
		lg: kasread.ListGet[FTPUserList, FTPUser]{
			Caller:    c,
			Action:    "get_ftpusers",
			Label:     "ftpuser",
			ArgName:   "login",
			FilterKey: "ftp_login",
			Decoder:   DecodeFTPUsers,
		},
		c: c,
	}
}

// List calls get_ftpusers without parameters and decodes the response
// into an FTPUserList covering every FTP user visible to the login.
func (c *Client) List(ctx context.Context) (FTPUserList, error) { return c.lg.List(ctx) }

// Get calls get_ftpusers with an ftp_login filter and returns the
// single matching FTPUser. The KAS API still wraps the result in an
// array; we unwrap it here so callers do not have to. An empty array
// surfaces as a not-found error.
func (c *Client) Get(ctx context.Context, login string) (FTPUser, error) {
	return c.lg.Get(ctx, login)
}

// DecodeFTPUsers maps the ReturnInfo of a get_ftpusers response (an
// Array of Maps) into the typed FTPUserList.
func DecodeFTPUsers(returnInfo soap.Value) (FTPUserList, error) {
	out, err := soap.DecodeArray(returnInfo, "ftpuser", func(item soap.Value) FTPUser {
		return FTPUser{
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
		}
	})
	if err != nil {
		return nil, err
	}
	return FTPUserList(out), nil
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
	return tablefmt.FieldValueHeaders
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
