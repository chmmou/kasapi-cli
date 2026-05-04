package domain

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/chmmou/kasapi-cli/internal/soap"
)

// Caller is the subset of *api.Client this package depends on. The
// indirection keeps tests free of network setup: a fake Caller can
// return a *soap.Response decoded from a fixture.
type Caller interface {
	Call(ctx context.Context, action string, params map[string]any) (*soap.Response, error)
}

// Domain is one entry of get_domains. The KAS list view exposes the
// account/server placement and a flattened SSL summary; the singular
// view (get_domains with a domain_name filter) omits the placement
// fields but adds dummy_host / dkim_selector and the full SSL cert
// PEM bodies. We model both shapes with one struct and rely on
// `omitempty` so neither view emits empty placeholder fields.
type Domain struct {
	Name           string `json:"domain_name" yaml:"domain_name"`
	TLD            string `json:"domain_tld,omitempty" yaml:"domain_tld,omitempty"`
	RedirectStatus int    `json:"domain_redirect_status" yaml:"domain_redirect_status"`
	Path           string `json:"domain_path" yaml:"domain_path"`
	Account        string `json:"domain_account,omitempty" yaml:"domain_account,omitempty"`
	Server         string `json:"domain_server,omitempty" yaml:"domain_server,omitempty"`

	DummyHost     string `json:"dummy_host,omitempty" yaml:"dummy_host,omitempty"`
	DKIMSelector  string `json:"dkim_selector,omitempty" yaml:"dkim_selector,omitempty"`
	FPSEActive    string `json:"fpse_active" yaml:"fpse_active"`
	PHPVersion    string `json:"php_version" yaml:"php_version"`
	PHPDeprecated string `json:"php_deprecated" yaml:"php_deprecated"`
	IsActive      string `json:"is_active" yaml:"is_active"`
	InProgress    string `json:"in_progress" yaml:"in_progress"`

	StatisticVersion  int    `json:"statistic_version" yaml:"statistic_version"`
	StatisticLanguage string `json:"statistic_language" yaml:"statistic_language"`

	SSL SSL `json:"ssl" yaml:"ssl"`
}

// SSL groups the ssl_* keys returned for a domain. The cert/key/CSR
// bodies are only present in the singular get_domains response
// (with domain_name filter); the list view leaves them as xsi:nil
// which decodes to an empty string here.
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

// DomainList is the typed payload of get_domains; satisfies cli.Tabular.
type DomainList []Domain

// TLD is one entry of get_topleveldomains.
type TLD struct {
	Name   string `json:"tld_name" yaml:"tld_name"`
	MinLen int    `json:"tld_minlen" yaml:"tld_minlen"`
	MaxLen int    `json:"tld_maxlen" yaml:"tld_maxlen"`
}

// TLDList is the typed payload of get_topleveldomains; satisfies
// cli.Tabular.
type TLDList []TLD

// Client groups the read endpoints scoped to domains:
// get_domains (list and singular) plus get_topleveldomains.
type Client struct {
	API Caller
}

// NewClient returns a Client backed by the given Caller.
func NewClient(c Caller) *Client { return &Client{API: c} }

// List calls get_domains without parameters and decodes the response
// into a DomainList covering every domain visible to the login.
func (c *Client) List(ctx context.Context) (DomainList, error) {
	resp, err := c.API.Call(ctx, "get_domains", nil)
	if err != nil {
		return nil, err
	}
	list, err := DecodeDomains(resp.Body.ReturnInfo)
	if err != nil {
		return nil, fmt.Errorf("domain: get_domains: %w", err)
	}
	return list, nil
}

// Get calls get_domains with a domain_name filter and returns the
// single matching Domain. The KAS API still wraps the result in an
// array; we unwrap it here so callers do not have to. An empty array
// surfaces as a not-found error.
func (c *Client) Get(ctx context.Context, name string) (Domain, error) {
	if name == "" {
		return Domain{}, fmt.Errorf("domain: name is required")
	}
	resp, err := c.API.Call(ctx, "get_domains", map[string]any{"domain_name": name})
	if err != nil {
		return Domain{}, err
	}
	list, err := DecodeDomains(resp.Body.ReturnInfo)
	if err != nil {
		return Domain{}, fmt.Errorf("domain: get_domains: %w", err)
	}
	if len(list) == 0 {
		return Domain{}, fmt.Errorf("domain: %q not found", name)
	}
	return list[0], nil
}

// TopLevelDomains calls get_topleveldomains and decodes the response.
func (c *Client) TopLevelDomains(ctx context.Context) (TLDList, error) {
	resp, err := c.API.Call(ctx, "get_topleveldomains", nil)
	if err != nil {
		return nil, err
	}
	list, err := DecodeTLDs(resp.Body.ReturnInfo)
	if err != nil {
		return nil, fmt.Errorf("domain: get_topleveldomains: %w", err)
	}
	return list, nil
}

// DecodeDomains maps the ReturnInfo of a get_domains response (an
// Array of Maps) into the typed DomainList.
func DecodeDomains(returnInfo soap.Value) (DomainList, error) {
	if returnInfo.Kind != soap.KindArray {
		return nil, fmt.Errorf("domain: expected ReturnInfo array, got kind %d", returnInfo.Kind)
	}
	out := make(DomainList, 0, len(returnInfo.Array))
	for i, item := range returnInfo.Array {
		if item.Kind != soap.KindMap {
			return nil, fmt.Errorf("domain: ReturnInfo[%d] is not a Map", i)
		}
		out = append(out, decodeDomain(item))
	}
	return out, nil
}

// DecodeTLDs maps the ReturnInfo of a get_topleveldomains response
// (an Array of Maps) into the typed TLDList.
func DecodeTLDs(returnInfo soap.Value) (TLDList, error) {
	if returnInfo.Kind != soap.KindArray {
		return nil, fmt.Errorf("domain: expected ReturnInfo array, got kind %d", returnInfo.Kind)
	}
	out := make(TLDList, 0, len(returnInfo.Array))
	for i, item := range returnInfo.Array {
		if item.Kind != soap.KindMap {
			return nil, fmt.Errorf("domain: ReturnInfo[%d] is not a Map", i)
		}
		out = append(out, TLD{
			Name:   getString(item, "tld_name"),
			MinLen: getInt(item, "tld_minlen"),
			MaxLen: getInt(item, "tld_maxlen"),
		})
	}
	return out, nil
}

func decodeDomain(m soap.Value) Domain {
	return Domain{
		Name:           getString(m, "domain_name"),
		TLD:            getString(m, "domain_tld"),
		RedirectStatus: getInt(m, "domain_redirect_status"),
		Path:           getString(m, "domain_path"),
		Account:        getString(m, "domain_account"),
		Server:         getString(m, "domain_server"),

		DummyHost:     getString(m, "dummy_host"),
		DKIMSelector:  getString(m, "dkim_selector"),
		FPSEActive:    getString(m, "fpse_active"),
		PHPVersion:    getString(m, "php_version"),
		PHPDeprecated: getString(m, "php_deprecated"),
		IsActive:      getString(m, "is_active"),
		InProgress:    getString(m, "in_progress"),

		StatisticVersion:  getInt(m, "statistic_version"),
		StatisticLanguage: getString(m, "statistic_language"),

		SSL: SSL{
			Proxy:         getString(m, "ssl_proxy"),
			CertificateIP: getString(m, "ssl_certificate_ip"),
			SNI:           getString(m, "ssl_certificate_sni"),
			SNIIsActive:   getString(m, "ssl_certificate_sni_is_active"),
			SNICSR:        getString(m, "ssl_certificate_sni_csr"),
			SNIKey:        getString(m, "ssl_certificate_sni_key"),
			SNICRT:        getString(m, "ssl_certificate_sni_crt"),
			SNIBundle:     getString(m, "ssl_certificate_sni_bundle"),
			SNIChainfile:  getString(m, "ssl_certificate_sni_chainfile"),
			SNIType:       getString(m, "ssl_certificate_sni_type"),
			SNIForceHTTPS: getString(m, "ssl_certificate_sni_force_https"),
			SNIHSTSMaxAge: getString(m, "ssl_certificate_sni_hsts_max_age"),
		},
	}
}

func getString(m soap.Value, key string) string {
	v, ok := m.Get(key)
	if !ok {
		return ""
	}
	return v.AsString()
}

func getInt(m soap.Value, key string) int {
	v, ok := m.Get(key)
	if !ok {
		return 0
	}
	switch v.Kind {
	case soap.KindInt:
		return int(v.Int)
	case soap.KindFloat:
		return int(v.Float)
	case soap.KindString:
		s := strings.TrimSpace(v.String)
		if s == "" {
			return 0
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return 0
		}
		return n
	}
	return 0
}

// TableHeaders returns the columns used by --output=table for
// DomainList.
func (DomainList) TableHeaders() []string {
	return []string{"DOMAIN", "ACCOUNT", "PHP", "SSL_TYPE", "ACTIVE", "IN_PROGRESS"}
}

// TableRows emits one row per Domain entry.
func (l DomainList) TableRows() [][]string {
	rows := make([][]string, 0, len(l))
	for _, d := range l {
		rows = append(rows, []string{
			d.Name,
			d.Account,
			d.PHPVersion,
			d.SSL.SNIType,
			d.IsActive,
			d.InProgress,
		})
	}
	return rows
}

// TableHeaders returns the columns used by --output=table for TLDList.
func (TLDList) TableHeaders() []string {
	return []string{"TLD", "MIN_LEN", "MAX_LEN"}
}

// TableRows emits one row per TLD entry.
func (l TLDList) TableRows() [][]string {
	rows := make([][]string, 0, len(l))
	for _, t := range l {
		rows = append(rows, []string{
			t.Name,
			strconv.Itoa(t.MinLen),
			strconv.Itoa(t.MaxLen),
		})
	}
	return rows
}

// TableHeaders for the singular Domain view: a key/value layout, since
// the record carries the SSL cert PEM blob and is too tall for a row.
func (Domain) TableHeaders() []string {
	return []string{"FIELD", "VALUE"}
}

// TableRows emits the scalar fields. The SSL cert PEM bodies are
// summarised; consumers that need them should use --output=json|yaml.
func (d Domain) TableRows() [][]string {
	rows := [][]string{
		{"domain_name", d.Name},
		{"domain_tld", d.TLD},
		{"domain_account", d.Account},
		{"domain_server", d.Server},
		{"domain_path", d.Path},
		{"domain_redirect_status", strconv.Itoa(d.RedirectStatus)},
		{"dummy_host", d.DummyHost},
		{"dkim_selector", d.DKIMSelector},
		{"fpse_active", d.FPSEActive},
		{"php_version", d.PHPVersion},
		{"php_deprecated", d.PHPDeprecated},
		{"is_active", d.IsActive},
		{"in_progress", d.InProgress},
		{"statistic_version", strconv.Itoa(d.StatisticVersion)},
		{"statistic_language", d.StatisticLanguage},
		{"ssl_proxy", d.SSL.Proxy},
		{"ssl_certificate_ip", d.SSL.CertificateIP},
		{"ssl_certificate_sni", d.SSL.SNI},
		{"ssl_certificate_sni_is_active", d.SSL.SNIIsActive},
		{"ssl_certificate_sni_type", d.SSL.SNIType},
		{"ssl_certificate_sni_force_https", d.SSL.SNIForceHTTPS},
		{"ssl_certificate_sni_hsts_max_age", d.SSL.SNIHSTSMaxAge},
		{"ssl_certificate_sni_csr", summarisePEM(d.SSL.SNICSR)},
		{"ssl_certificate_sni_key", summarisePEM(d.SSL.SNIKey)},
		{"ssl_certificate_sni_crt", summarisePEM(d.SSL.SNICRT)},
		{"ssl_certificate_sni_bundle", summarisePEM(d.SSL.SNIBundle)},
		{"ssl_certificate_sni_chainfile", summarisePEM(d.SSL.SNIChainfile)},
	}
	return rows
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
