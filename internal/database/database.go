package database

import (
	"context"
	"fmt"
	"strconv"

	"github.com/chmmou/kasapi-cli/internal/soap"
)

// Caller is the subset of *api.Client this package depends on. The
// indirection keeps tests free of network setup: a fake Caller can
// return a *soap.Response decoded from a fixture.
type Caller interface {
	Call(ctx context.Context, action string, params map[string]any) (*soap.Response, error)
}

// Database is one entry of get_databases. The list and singular views
// (the latter being get_databases called with a database_login filter)
// return the same Map shape, so a single struct covers both.
type Database struct {
	Name              string  `json:"database_name" yaml:"database_name"`
	Login             string  `json:"database_login" yaml:"database_login"`
	Password          string  `json:"database_password,omitempty" yaml:"database_password,omitempty"`
	Comment           string  `json:"database_comment" yaml:"database_comment"`
	AllowedHosts      string  `json:"database_allowed_hosts" yaml:"database_allowed_hosts"`
	UsedDatabaseSpace float64 `json:"used_database_space" yaml:"used_database_space"`
}

// DatabaseList is the typed payload of get_databases; satisfies
// cli.Tabular.
type DatabaseList []Database

// Client groups the read endpoints scoped to databases:
// get_databases (list and singular).
type Client struct {
	API Caller
}

// NewClient returns a Client backed by the given Caller.
func NewClient(c Caller) *Client { return &Client{API: c} }

// List calls get_databases without parameters and decodes the response
// into a DatabaseList covering every database visible to the login.
func (c *Client) List(ctx context.Context) (DatabaseList, error) {
	resp, err := c.API.Call(ctx, "get_databases", nil)
	if err != nil {
		return nil, err
	}
	list, err := DecodeDatabases(resp.Body.ReturnInfo)
	if err != nil {
		return nil, fmt.Errorf("database: get_databases: %w", err)
	}
	return list, nil
}

// Get calls get_databases with a database_login filter and returns the
// single matching Database. The KAS API still wraps the result in an
// array; we unwrap it here so callers do not have to. An empty array
// surfaces as a not-found error.
func (c *Client) Get(ctx context.Context, login string) (Database, error) {
	if login == "" {
		return Database{}, fmt.Errorf("database: login is required")
	}
	resp, err := c.API.Call(ctx, "get_databases", map[string]any{"database_login": login})
	if err != nil {
		return Database{}, err
	}
	list, err := DecodeDatabases(resp.Body.ReturnInfo)
	if err != nil {
		return Database{}, fmt.Errorf("database: get_databases: %w", err)
	}
	if len(list) == 0 {
		return Database{}, fmt.Errorf("database: %q not found", login)
	}
	return list[0], nil
}

// DecodeDatabases maps the ReturnInfo of a get_databases response (an
// Array of Maps) into the typed DatabaseList.
func DecodeDatabases(returnInfo soap.Value) (DatabaseList, error) {
	out, err := soap.DecodeArray(returnInfo, "database", func(item soap.Value) Database {
		return Database{
			Name:              item.MapString("database_name"),
			Login:             item.MapString("database_login"),
			Password:          item.MapString("database_password"),
			Comment:           item.MapString("database_comment"),
			AllowedHosts:      item.MapString("database_allowed_hosts"),
			UsedDatabaseSpace: item.MapFloat("used_database_space"),
		}
	})
	if err != nil {
		return nil, err
	}
	return DatabaseList(out), nil
}

// TableHeaders returns the columns used by --output=table for
// DatabaseList.
func (DatabaseList) TableHeaders() []string {
	return []string{"LOGIN", "NAME", "COMMENT", "ALLOWED_HOSTS", "USED_MB"}
}

// TableRows emits one row per Database entry. used_database_space is
// reported in KiB by KAS; we convert to MB to match the units used in
// the accounts/mailaccounts list views.
func (l DatabaseList) TableRows() [][]string {
	rows := make([][]string, 0, len(l))
	for _, d := range l {
		rows = append(rows, []string{
			d.Login,
			d.Name,
			d.Comment,
			d.AllowedHosts,
			strconv.FormatFloat(d.UsedDatabaseSpace/1024, 'f', 2, 64),
		})
	}
	return rows
}

// TableHeaders for the singular Database view: a key/value layout to
// match the rest of the singular detail commands (mail accounts, accounts).
func (Database) TableHeaders() []string {
	return []string{"FIELD", "VALUE"}
}

// TableRows emits the scalar fields. database_password is intentionally
// omitted — consumers that need it should use --output=json|yaml.
func (d Database) TableRows() [][]string {
	return [][]string{
		{"database_login", d.Login},
		{"database_name", d.Name},
		{"database_comment", d.Comment},
		{"database_allowed_hosts", d.AllowedHosts},
		{"used_database_space", strconv.FormatFloat(d.UsedDatabaseSpace/1024, 'f', 2, 64) + " MB"},
	}
}
