package softwareinstall

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

// SoftwareInstall is one entry of get_softwareinstall (note: the KAS
// action name is singular for both the list and the filtered variant).
// Each entry describes one installable software package: ID, name,
// version, runtime requirements (PHP/MySQL/MariaDB version bands),
// and whether the current account can install it.
//
// The KAS API returns `image` as a base64-encoded data URI of the
// software's logo; we keep it on the struct for JSON/YAML round-trip
// fidelity but strip it from the table views since it would dwarf
// every other column.
type SoftwareInstall struct {
	ID          string `json:"software_id" yaml:"software_id"`
	Name        string `json:"software_name" yaml:"software_name"`
	Category    string `json:"software_category" yaml:"software_category"`
	Version     string `json:"software_version" yaml:"software_version"`
	Licence     string `json:"software_licence" yaml:"software_licence"`
	Description string `json:"description" yaml:"description"`
	Image       string `json:"image" yaml:"image"`

	HasExampleData string `json:"software_has_example_data" yaml:"software_has_example_data"`

	// Version constraints. KAS sends them as strings ("8.4", "0.0",
	// "10.5", …) where 0.0 effectively means "not applicable" — we
	// preserve the wire form rather than parsing them.
	PHPVersion         string `json:"software_version_php" yaml:"software_version_php"`
	PHPVersionUpto     string `json:"software_version_php_upto" yaml:"software_version_php_upto"`
	PHPHtaccess        string `json:"software_version_php_htaccess" yaml:"software_version_php_htaccess"`
	PHPWantCGI         string `json:"software_version_php_want_cgi" yaml:"software_version_php_want_cgi"`
	MySQLVersion       string `json:"software_version_mysql" yaml:"software_version_mysql"`
	MySQLVersionUpto   string `json:"software_version_mysql_upto" yaml:"software_version_mysql_upto"`
	MariaDBVersion     string `json:"software_version_mariadb" yaml:"software_version_mariadb"`
	MariaDBVersionUpto string `json:"software_version_mariadb_upto" yaml:"software_version_mariadb_upto"`

	CanBeInstalled string `json:"software_can_be_installed" yaml:"software_can_be_installed"`
	CanBeMessage   string `json:"software_can_be_message" yaml:"software_can_be_message"`
}

// SoftwareInstallList is the typed payload of get_softwareinstall;
// satisfies cli.Tabular.
type SoftwareInstallList []SoftwareInstall

// Client groups the read endpoint scoped to software-install
// templates: get_softwareinstall (list and singular).
type Client struct {
	API Caller
}

// NewClient returns a Client backed by the given Caller.
func NewClient(c Caller) *Client { return &Client{API: c} }

// List calls get_softwareinstall without parameters and decodes the
// response into a SoftwareInstallList covering every installable
// software package visible to the login.
func (c *Client) List(ctx context.Context) (SoftwareInstallList, error) {
	resp, err := c.API.Call(ctx, "get_softwareinstall", nil)
	if err != nil {
		return nil, err
	}
	list, err := DecodeSoftwareInstalls(resp.Body.ReturnInfo)
	if err != nil {
		return nil, fmt.Errorf("softwareinstall: get_softwareinstall: %w", err)
	}
	return list, nil
}

// Get calls get_softwareinstall with a software_id filter and returns
// the single matching SoftwareInstall. The KAS API still wraps the
// result in an array; we unwrap it here so callers do not have to.
// An empty array surfaces as a not-found error.
func (c *Client) Get(ctx context.Context, id string) (SoftwareInstall, error) {
	if id == "" {
		return SoftwareInstall{}, fmt.Errorf("softwareinstall: id is required")
	}
	resp, err := c.API.Call(ctx, "get_softwareinstall", map[string]any{"software_id": id})
	if err != nil {
		return SoftwareInstall{}, err
	}
	list, err := DecodeSoftwareInstalls(resp.Body.ReturnInfo)
	if err != nil {
		return SoftwareInstall{}, fmt.Errorf("softwareinstall: get_softwareinstall: %w", err)
	}
	if len(list) == 0 {
		return SoftwareInstall{}, fmt.Errorf("softwareinstall: %q not found", id)
	}
	return list[0], nil
}

// DecodeSoftwareInstalls maps the ReturnInfo of a get_softwareinstall
// response (an Array of Maps) into the typed SoftwareInstallList.
func DecodeSoftwareInstalls(returnInfo soap.Value) (SoftwareInstallList, error) {
	if returnInfo.Kind != soap.KindArray {
		return nil, fmt.Errorf("softwareinstall: expected ReturnInfo array, got kind %d", returnInfo.Kind)
	}
	out := make(SoftwareInstallList, 0, len(returnInfo.Array))
	for i, item := range returnInfo.Array {
		if item.Kind != soap.KindMap {
			return nil, fmt.Errorf("softwareinstall: ReturnInfo[%d] is not a Map", i)
		}
		out = append(out, SoftwareInstall{
			ID:                 getString(item, "software_id"),
			Name:               getString(item, "software_name"),
			Category:           getString(item, "software_category"),
			Version:            getString(item, "software_version"),
			Licence:            getString(item, "software_licence"),
			Description:        getString(item, "description"),
			Image:              getString(item, "image"),
			HasExampleData:     getString(item, "software_has_example_data"),
			PHPVersion:         getString(item, "software_version_php"),
			PHPVersionUpto:     getString(item, "software_version_php_upto"),
			PHPHtaccess:        getString(item, "software_version_php_htaccess"),
			PHPWantCGI:         getString(item, "software_version_php_want_cgi"),
			MySQLVersion:       getString(item, "software_version_mysql"),
			MySQLVersionUpto:   getString(item, "software_version_mysql_upto"),
			MariaDBVersion:     getString(item, "software_version_mariadb"),
			MariaDBVersionUpto: getString(item, "software_version_mariadb_upto"),
			CanBeInstalled:     getString(item, "software_can_be_installed"),
			CanBeMessage:       getString(item, "software_can_be_message"),
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
// SoftwareInstallList. The image and description fields are omitted
// — the former because it is a multi-kB base64 blob, the latter
// because it can wrap across many lines; both stay accessible via
// --output=json|yaml.
func (SoftwareInstallList) TableHeaders() []string {
	return []string{"ID", "NAME", "CATEGORY", "VERSION", "PHP", "DB", "INSTALLABLE"}
}

// TableRows emits one row per SoftwareInstall entry. The PHP and DB
// columns collapse the {min, upto} pair so the table stays narrow;
// 0.0 sentinels are kept verbatim because they signal "not
// applicable" on the wire.
func (l SoftwareInstallList) TableRows() [][]string {
	rows := make([][]string, 0, len(l))
	for _, s := range l {
		rows = append(rows, []string{
			s.ID,
			s.Name,
			s.Category,
			s.Version,
			versionRange(s.PHPVersion, s.PHPVersionUpto),
			dbRange(s.MariaDBVersion, s.MariaDBVersionUpto, s.MySQLVersion, s.MySQLVersionUpto),
			s.CanBeInstalled,
		})
	}
	return rows
}

// versionRange formats a {from, to} pair as "from..to" or just "from"
// when the two coincide. An empty/zero from returns "—".
func versionRange(from, to string) string {
	if from == "" || from == "0.0" {
		return "—"
	}
	if to == "" || to == from {
		return from
	}
	return from + ".." + to
}

// dbRange picks MariaDB if it has a non-zero range (the common case
// on All-Inkl) and falls back to MySQL otherwise. The leading prefix
// makes it explicit which engine the version refers to.
func dbRange(mariaFrom, mariaTo, mysqlFrom, mysqlTo string) string {
	if r := versionRange(mariaFrom, mariaTo); r != "—" {
		return "MariaDB " + r
	}
	if r := versionRange(mysqlFrom, mysqlTo); r != "—" {
		return "MySQL " + r
	}
	return "—"
}

// TableHeaders for the singular SoftwareInstall view: a key/value
// layout. The image data URI is intentionally omitted to keep the
// output usable in a terminal — consumers that need the logo should
// use --output=json|yaml.
func (SoftwareInstall) TableHeaders() []string {
	return []string{"FIELD", "VALUE"}
}

// TableRows emits the scalar fields. image is omitted (see
// TableHeaders); software_can_be_message appears immediately after
// software_can_be_installed because the message explains a "N" value.
func (s SoftwareInstall) TableRows() [][]string {
	return [][]string{
		{"software_id", s.ID},
		{"software_name", s.Name},
		{"software_category", s.Category},
		{"software_version", s.Version},
		{"software_licence", s.Licence},
		{"software_has_example_data", s.HasExampleData},
		{"software_version_php", s.PHPVersion},
		{"software_version_php_upto", s.PHPVersionUpto},
		{"software_version_php_htaccess", s.PHPHtaccess},
		{"software_version_php_want_cgi", s.PHPWantCGI},
		{"software_version_mysql", s.MySQLVersion},
		{"software_version_mysql_upto", s.MySQLVersionUpto},
		{"software_version_mariadb", s.MariaDBVersion},
		{"software_version_mariadb_upto", s.MariaDBVersionUpto},
		{"software_can_be_installed", s.CanBeInstalled},
		{"software_can_be_message", s.CanBeMessage},
		{"description", s.Description},
	}
}
