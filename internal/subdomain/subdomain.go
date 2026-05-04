package subdomain

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

// Subdomain is one entry of get_subdomains. The shape mirrors the
// domain list view (same SSL summary, same lifecycle flags) but is
// keyed on subdomain_* fields.
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

// Client groups the read endpoints scoped to subdomains: get_subdomains.
type Client struct {
	API Caller
}

// NewClient returns a Client backed by the given Caller.
func NewClient(c Caller) *Client { return &Client{API: c} }

// List calls get_subdomains and decodes the response into a
// SubdomainList. The KAS API accepts no parameters on this endpoint.
func (c *Client) List(ctx context.Context) (SubdomainList, error) {
	resp, err := c.API.Call(ctx, "get_subdomains", nil)
	if err != nil {
		return nil, err
	}
	list, err := DecodeSubdomains(resp.Body.ReturnInfo)
	if err != nil {
		return nil, fmt.Errorf("subdomain: get_subdomains: %w", err)
	}
	return list, nil
}

// DecodeSubdomains maps the ReturnInfo of a get_subdomains response
// (an Array of Maps) into the typed SubdomainList.
func DecodeSubdomains(returnInfo soap.Value) (SubdomainList, error) {
	if returnInfo.Kind != soap.KindArray {
		return nil, fmt.Errorf("subdomain: expected ReturnInfo array, got kind %d", returnInfo.Kind)
	}
	out := make(SubdomainList, 0, len(returnInfo.Array))
	for i, item := range returnInfo.Array {
		if item.Kind != soap.KindMap {
			return nil, fmt.Errorf("subdomain: ReturnInfo[%d] is not a Map", i)
		}
		out = append(out, Subdomain{
			Name:           getString(item, "subdomain_name"),
			RedirectStatus: getInt(item, "subdomain_redirect_status"),
			Path:           getString(item, "subdomain_path"),
			Account:        getString(item, "subdomain_account"),
			Server:         getString(item, "subdomain_server"),

			FPSEActive:    getString(item, "fpse_active"),
			PHPVersion:    getString(item, "php_version"),
			PHPDeprecated: getString(item, "php_deprecated"),
			IsActive:      getString(item, "is_active"),
			InProgress:    getString(item, "in_progress"),

			StatisticVersion:  getInt(item, "statistic_version"),
			StatisticLanguage: getString(item, "statistic_language"),

			SSL: SSL{
				Proxy:         getString(item, "ssl_proxy"),
				CertificateIP: getString(item, "ssl_certificate_ip"),
				SNI:           getString(item, "ssl_certificate_sni"),
				SNIIsActive:   getString(item, "ssl_certificate_sni_is_active"),
				SNICSR:        getString(item, "ssl_certificate_sni_csr"),
				SNIKey:        getString(item, "ssl_certificate_sni_key"),
				SNICRT:        getString(item, "ssl_certificate_sni_crt"),
				SNIBundle:     getString(item, "ssl_certificate_sni_bundle"),
				SNIChainfile:  getString(item, "ssl_certificate_sni_chainfile"),
				SNIType:       getString(item, "ssl_certificate_sni_type"),
				SNIForceHTTPS: getString(item, "ssl_certificate_sni_force_https"),
				SNIHSTSMaxAge: getString(item, "ssl_certificate_sni_hsts_max_age"),
			},
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
