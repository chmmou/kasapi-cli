package directoryprotection

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

// DirectoryProtection is one (path, user) tuple of the KAS htaccess /
// directory-protection table. The KAS API returns one entry per user
// authorised on a given directory, so a single protected path with
// three users yields three entries — `Get`-style unwrap to a single
// struct is therefore not appropriate; this module exposes the list
// shape and lets the caller filter by `directory_path`.
type DirectoryProtection struct {
	User       string `json:"directory_user" yaml:"directory_user"`
	Path       string `json:"directory_path" yaml:"directory_path"`
	AuthName   string `json:"directory_authname" yaml:"directory_authname"`
	Password   string `json:"directory_password,omitempty" yaml:"directory_password,omitempty"`
	InProgress string `json:"in_progress" yaml:"in_progress"`
}

// DirectoryProtectionList is the typed payload of get_directoryprotection;
// satisfies cli.Tabular.
type DirectoryProtectionList []DirectoryProtection

// Client groups the read endpoint scoped to directory protection.
//
// Both the unfiltered and `directory_path`-filtered variants share
// the same KAS action name `get_directoryprotection` (note: singular,
// even when no filter is set), so this module exposes one List
// method that takes an optional path.
type Client struct {
	API Caller
}

// NewClient returns a Client backed by the given Caller.
func NewClient(c Caller) *Client { return &Client{API: c} }

// List calls get_directoryprotection and decodes the response into a
// DirectoryProtectionList. An empty path returns every protected
// directory; a non-empty path filters to that directory's user
// entries (still a list, since multiple users per path are possible).
func (c *Client) List(ctx context.Context, path string) (DirectoryProtectionList, error) {
	var params map[string]any
	if path != "" {
		params = map[string]any{"directory_path": path}
	}
	resp, err := c.API.Call(ctx, "get_directoryprotection", params)
	if err != nil {
		return nil, err
	}
	list, err := DecodeDirectoryProtections(resp.Body.ReturnInfo)
	if err != nil {
		return nil, fmt.Errorf("directoryprotection: get_directoryprotection: %w", err)
	}
	return list, nil
}

// DecodeDirectoryProtections maps the ReturnInfo of a
// get_directoryprotection response (an Array of Maps) into the typed
// DirectoryProtectionList.
func DecodeDirectoryProtections(returnInfo soap.Value) (DirectoryProtectionList, error) {
	if returnInfo.Kind != soap.KindArray {
		return nil, fmt.Errorf("directoryprotection: expected ReturnInfo array, got kind %d", returnInfo.Kind)
	}
	out := make(DirectoryProtectionList, 0, len(returnInfo.Array))
	for i, item := range returnInfo.Array {
		if item.Kind != soap.KindMap {
			return nil, fmt.Errorf("directoryprotection: ReturnInfo[%d] is not a Map", i)
		}
		out = append(out, DirectoryProtection{
			User:       item.MapString("directory_user"),
			Path:       item.MapString("directory_path"),
			AuthName:   item.MapString("directory_authname"),
			Password:   item.MapString("directory_password"),
			InProgress: item.MapString("in_progress"),
		})
	}
	return out, nil
}

// TableHeaders returns the columns used by --output=table for
// DirectoryProtectionList.
func (DirectoryProtectionList) TableHeaders() []string {
	return []string{"PATH", "USER", "AUTHNAME", "IN_PROGRESS"}
}

// TableRows emits one row per (path, user) entry. directory_password
// is intentionally omitted — consumers that need it should use
// --output=json|yaml.
func (l DirectoryProtectionList) TableRows() [][]string {
	rows := make([][]string, 0, len(l))
	for _, d := range l {
		rows = append(rows, []string{
			d.Path,
			d.User,
			d.AuthName,
			d.InProgress,
		})
	}
	return rows
}
