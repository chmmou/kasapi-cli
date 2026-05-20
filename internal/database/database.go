package database

import (
	"context"
	"strconv"

	"github.com/chmmou/kasapi-cli/internal/kasread"
	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/tablefmt"
)

// Caller is the subset of *api.Client this package depends on. The
// indirection keeps tests free of network setup: a fake Caller can
// return a *soap.Response decoded from a fixture.
type Caller = kasread.Caller

// The KAS API encodes the in_progress async-write flag as the literal
// strings "TRUE" / "FALSE" (not booleans). Both literals are exported
// so mappings + tests share a single source of truth instead of
// re-typing string literals at every comparison site.
const (
	InProgressFalse = "FALSE"
	InProgressTrue  = "TRUE"
)

// Database is one entry of get_databases. The list and singular views
// (the latter being get_databases called with a database_login filter)
// return the same Map shape, so a single struct covers both.
//
// in_progress is a pending-async-write flag the KAS API surfaces on
// every database row (typically "FALSE"). It is rendered without
// omitempty for parity with the majority of the read modules
// (account, mailaccount, mailinglist, sambauser, ftpuser, …) — the
// fixture has shown it on every row captured so far, and a leak as
// the empty string is harmless if the API ever stops sending it.
type Database struct {
	Name              string  `json:"database_name" yaml:"database_name"`
	Login             string  `json:"database_login" yaml:"database_login"`
	Password          string  `json:"database_password,omitempty" yaml:"database_password,omitempty"`
	Comment           string  `json:"database_comment" yaml:"database_comment"`
	AllowedHosts      string  `json:"database_allowed_hosts" yaml:"database_allowed_hosts"`
	UsedDatabaseSpace float64 `json:"used_database_space" yaml:"used_database_space"`
	InProgress        string  `json:"in_progress" yaml:"in_progress"`
}

// DatabaseList is the typed payload of get_databases; satisfies
// cli.Tabular.
type DatabaseList []Database

// Client groups the read endpoints scoped to databases
// (get_databases, list and singular) and the write endpoints
// add_database / update_database / delete_database (see write.go). The
// raw Caller is kept alongside the read helper so the write methods
// can dispatch their own KAS actions through the shared kaswrite seam.
type Client struct {
	lg kasread.ListGet[DatabaseList, Database]
	c  Caller
}

// NewClient returns a Client backed by the given Caller.
func NewClient(c Caller) *Client {
	return &Client{
		lg: kasread.ListGet[DatabaseList, Database]{
			Caller:    c,
			Action:    "get_databases",
			Label:     "database",
			ArgName:   "login",
			FilterKey: "database_login",
			Decoder:   DecodeDatabases,
		},
		c: c,
	}
}

// List calls get_databases without parameters and decodes the response
// into a DatabaseList covering every database visible to the login.
func (c *Client) List(ctx context.Context) (DatabaseList, error) { return c.lg.List(ctx) }

// Get calls get_databases with a database_login filter and returns the
// single matching Database. The KAS API still wraps the result in an
// array; we unwrap it here so callers do not have to. An empty array
// surfaces as a not-found error.
func (c *Client) Get(ctx context.Context, login string) (Database, error) {
	return c.lg.Get(ctx, login)
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
			InProgress:        item.MapString("in_progress"),
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
	return []string{"LOGIN", "NAME", "COMMENT", "ALLOWED_HOSTS", "USED", "IN_PROGRESS"}
}

// TableRows emits one row per Database entry. used_database_space is
// reported in KiB by KAS; we convert to MB and render the unit as part
// of the value so the list cells share a single convention with the
// singular FIELD/VALUE detail view (both carry the unit in the value,
// not in a header column).
func (l DatabaseList) TableRows() [][]string {
	rows := make([][]string, 0, len(l))
	for _, d := range l {
		rows = append(rows, []string{
			d.Login,
			d.Name,
			d.Comment,
			d.AllowedHosts,
			strconv.FormatFloat(d.UsedDatabaseSpace/1024, 'f', 2, 64) + " MB",
			d.InProgress,
		})
	}
	return rows
}

// TableHeaders for the singular Database view: a key/value layout to
// match the rest of the singular detail commands (mail accounts, accounts).
func (Database) TableHeaders() []string {
	return tablefmt.FieldValueHeaders
}

// TableRows emits the scalar fields. database_password is intentionally
// omitted — consumers that need it should use --output=json|yaml. The
// used_database_space row carries the unit (" MB") as part of the
// value so the singular and list views share a single convention.
// in_progress only appears when the API actually returned it.
func (d Database) TableRows() [][]string {
	rows := [][]string{
		{"database_login", d.Login},
		{"database_name", d.Name},
		{"database_comment", d.Comment},
		{"database_allowed_hosts", d.AllowedHosts},
		{"used_database_space", strconv.FormatFloat(d.UsedDatabaseSpace/1024, 'f', 2, 64) + " MB"},
	}
	if d.InProgress != "" {
		rows = append(rows, []string{"in_progress", d.InProgress})
	}
	return rows
}
