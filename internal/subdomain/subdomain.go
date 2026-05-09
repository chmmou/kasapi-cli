package subdomain

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/chmmou/kasapi-cli/internal/kasread"
	"github.com/chmmou/kasapi-cli/internal/soap"
)

// Caller is the subset of *api.Client this package depends on. The
// indirection keeps tests free of network setup: a fake Caller can
// return a *soap.Response decoded from a fixture.
type Caller = kasread.Caller

// Subdomain is one entry of get_subdomains. The KAS list view exposes
// the account/server placement and a flattened SSL summary; the
// singular view (get_subdomains with a subdomain_name filter) returns
// the same Map shape but additionally fills the SSL cert/key/CSR PEM
// bodies that the list view leaves as xsi:nil. We model both shapes
// with one struct and rely on `omitempty` for the cert bodies.
type Subdomain struct {
	Name           string `json:"subdomain_name" yaml:"subdomain_name"`
	RedirectStatus int    `json:"subdomain_redirect_status" yaml:"subdomain_redirect_status"`
	Path           string `json:"subdomain_path" yaml:"subdomain_path"`
	Account        string `json:"subdomain_account" yaml:"subdomain_account"`
	Server         string `json:"subdomain_server" yaml:"subdomain_server"`

	FPSEActive    string `json:"fpse_active" yaml:"fpse_active"`
	PHPVersion    string `json:"php_version" yaml:"php_version"`
	PHPDeprecated string `json:"php_deprecated" yaml:"php_deprecated"`
	IsActive      string `json:"is_active" yaml:"is_active"`
	InProgress    string `json:"in_progress" yaml:"in_progress"`

	StatisticVersion  int    `json:"statistic_version" yaml:"statistic_version"`
	StatisticLanguage string `json:"statistic_language" yaml:"statistic_language"`

	SSL SSL `json:"ssl" yaml:"ssl"`
}

// SSL groups the ssl_* keys returned for a subdomain. The list view
// leaves the cert/key/CSR bodies as xsi:nil, decoded here as the
// empty string.
type SSL struct {
	Proxy         string `json:"proxy" yaml:"proxy"`
	CertificateIP string `json:"certificate_ip" yaml:"certificate_ip"`
	SNI           string `json:"sni" yaml:"sni"`
	SNIIsActive   string `json:"sni_is_active" yaml:"sni_is_active"`
	SNICSR        string `json:"sni_csr,omitempty" yaml:"sni_csr,omitempty"`
	SNIKey        string `json:"sni_key,omitempty" yaml:"sni_key,omitempty"`
	SNICRT        string `json:"sni_crt,omitempty" yaml:"sni_crt,omitempty"`
	SNIBundle     string `json:"sni_bundle,omitempty" yaml:"sni_bundle,omitempty"`
	SNIChainfile  string `json:"sni_chainfile,omitempty" yaml:"sni_chainfile,omitempty"`
	SNIType       string `json:"sni_type,omitempty" yaml:"sni_type,omitempty"`
	SNIForceHTTPS string `json:"sni_force_https,omitempty" yaml:"sni_force_https,omitempty"`
	SNIHSTSMaxAge string `json:"sni_hsts_max_age,omitempty" yaml:"sni_hsts_max_age,omitempty"`
}

// SubdomainList is the typed payload of get_subdomains; satisfies
// cli.Tabular.
type SubdomainList []Subdomain

// Client groups the read endpoints scoped to subdomains:
// get_subdomains (list and singular).
type Client struct {
	lg kasread.ListGet[SubdomainList, Subdomain]
}

// NewClient returns a Client backed by the given Caller.
func NewClient(c Caller) *Client {
	return &Client{lg: kasread.ListGet[SubdomainList, Subdomain]{
		Caller:    c,
		Action:    "get_subdomains",
		Label:     "subdomain",
		ArgName:   "name",
		FilterKey: "subdomain_name",
		Decoder:   DecodeSubdomains,
	}}
}

// List calls get_subdomains without parameters and decodes the
// response into a SubdomainList covering every subdomain visible to
// the login.
func (c *Client) List(ctx context.Context) (SubdomainList, error) { return c.lg.List(ctx) }

// Get calls get_subdomains with a subdomain_name filter and returns
// the single matching Subdomain. The KAS API still wraps the result
// in an array; we unwrap it here so callers do not have to. An empty
// array surfaces as a not-found error.
func (c *Client) Get(ctx context.Context, name string) (Subdomain, error) {
	return c.lg.Get(ctx, name)
}

// DecodeSubdomains maps the ReturnInfo of a get_subdomains response
// (an Array of Maps) into the typed SubdomainList.
func DecodeSubdomains(returnInfo soap.Value) (SubdomainList, error) {
	out, err := soap.DecodeArray(returnInfo, "subdomain", func(item soap.Value) Subdomain {
		return Subdomain{
			Name:           item.MapString("subdomain_name"),
			RedirectStatus: item.MapInt("subdomain_redirect_status"),
			Path:           item.MapString("subdomain_path"),
			Account:        item.MapString("subdomain_account"),
			Server:         item.MapString("subdomain_server"),

			FPSEActive:    item.MapString("fpse_active"),
			PHPVersion:    item.MapString("php_version"),
			PHPDeprecated: item.MapString("php_deprecated"),
			IsActive:      item.MapString("is_active"),
			InProgress:    item.MapString("in_progress"),

			StatisticVersion:  item.MapInt("statistic_version"),
			StatisticLanguage: item.MapString("statistic_language"),

			SSL: SSL{
				Proxy:         item.MapString("ssl_proxy"),
				CertificateIP: item.MapString("ssl_certificate_ip"),
				SNI:           item.MapString("ssl_certificate_sni"),
				SNIIsActive:   item.MapString("ssl_certificate_sni_is_active"),
				SNICSR:        item.MapString("ssl_certificate_sni_csr"),
				SNIKey:        item.MapString("ssl_certificate_sni_key"),
				SNICRT:        item.MapString("ssl_certificate_sni_crt"),
				SNIBundle:     item.MapString("ssl_certificate_sni_bundle"),
				SNIChainfile:  item.MapString("ssl_certificate_sni_chainfile"),
				SNIType:       item.MapString("ssl_certificate_sni_type"),
				SNIForceHTTPS: item.MapString("ssl_certificate_sni_force_https"),
				SNIHSTSMaxAge: item.MapString("ssl_certificate_sni_hsts_max_age"),
			},
		}
	})
	if err != nil {
		return nil, err
	}
	return SubdomainList(out), nil
}

// TableHeaders returns the columns used by --output=table for
// SubdomainList.
func (SubdomainList) TableHeaders() []string {
	return []string{"SUBDOMAIN", "ACCOUNT", "PHP", "SSL_TYPE", "ACTIVE", "IN_PROGRESS"}
}

// TableRows emits one row per Subdomain entry.
func (l SubdomainList) TableRows() [][]string {
	rows := make([][]string, 0, len(l))
	for _, s := range l {
		rows = append(rows, []string{
			s.Name,
			s.Account,
			s.PHPVersion,
			s.SSL.SNIType,
			s.IsActive,
			s.InProgress,
		})
	}
	return rows
}

// TableHeaders for the singular Subdomain view: a key/value layout,
// since the record may carry an SSL cert PEM blob and is too tall for
// a row.
func (Subdomain) TableHeaders() []string {
	return []string{"FIELD", "VALUE"}
}

// TableRows emits the scalar fields. The SSL cert PEM bodies are
// summarised; consumers that need them should use --output=json|yaml.
func (s Subdomain) TableRows() [][]string {
	return [][]string{
		{"subdomain_name", s.Name},
		{"subdomain_account", s.Account},
		{"subdomain_server", s.Server},
		{"subdomain_path", s.Path},
		{"subdomain_redirect_status", strconv.Itoa(s.RedirectStatus)},
		{"fpse_active", s.FPSEActive},
		{"php_version", s.PHPVersion},
		{"php_deprecated", s.PHPDeprecated},
		{"is_active", s.IsActive},
		{"in_progress", s.InProgress},
		{"statistic_version", strconv.Itoa(s.StatisticVersion)},
		{"statistic_language", s.StatisticLanguage},
		{"ssl_proxy", s.SSL.Proxy},
		{"ssl_certificate_ip", s.SSL.CertificateIP},
		{"ssl_certificate_sni", s.SSL.SNI},
		{"ssl_certificate_sni_is_active", s.SSL.SNIIsActive},
		{"ssl_certificate_sni_type", s.SSL.SNIType},
		{"ssl_certificate_sni_force_https", s.SSL.SNIForceHTTPS},
		{"ssl_certificate_sni_hsts_max_age", s.SSL.SNIHSTSMaxAge},
		{"ssl_certificate_sni_csr", summarisePEM(s.SSL.SNICSR)},
		{"ssl_certificate_sni_key", summarisePEM(s.SSL.SNIKey)},
		{"ssl_certificate_sni_crt", summarisePEM(s.SSL.SNICRT)},
		{"ssl_certificate_sni_bundle", summarisePEM(s.SSL.SNIBundle)},
		{"ssl_certificate_sni_chainfile", summarisePEM(s.SSL.SNIChainfile)},
	}
}

// summarisePEM collapses a multi-line PEM blob to a single line marker
// so the key/value table stays readable. Empty input passes through
// unchanged.
func summarisePEM(s string) string {
	if s == "" {
		return ""
	}
	if !strings.Contains(s, "\n") {
		return s
	}
	return fmt.Sprintf("<%d bytes, %d lines>", len(s), strings.Count(s, "\n")+1)
}
