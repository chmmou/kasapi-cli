package server

import (
	"context"
	"fmt"

	"github.com/chmmou/kasapi-cli/internal/kasread"
	"github.com/chmmou/kasapi-cli/internal/soap"
)

// Caller is the subset of *api.Client this package depends on.
type Caller = kasread.Caller

// Service is one entry from get_server_information. The KAS API
// returns a heterogeneous list: mysql carries a version_type, php
// entries carry interface and file_extension, the os entry carries a
// distribution. We keep all known fields as optional strings so the
// struct survives KAS adding new service kinds.
type Service struct {
	Service       string `json:"service" yaml:"service"`
	Version       string `json:"version" yaml:"version"`
	VersionType   string `json:"version_type,omitempty" yaml:"version_type,omitempty"`
	Interface     string `json:"interface,omitempty" yaml:"interface,omitempty"`
	FileExtension string `json:"file_extension,omitempty" yaml:"file_extension,omitempty"`
	Distribution  string `json:"distribution,omitempty" yaml:"distribution,omitempty"`
}

// ServiceList is the typed payload of get_server_information; it
// satisfies cli.Tabular so --output=table works without a special
// renderer.
type ServiceList []Service

// Client groups the read endpoints scoped to the host server.
type Client struct {
	API Caller
}

// NewClient returns a Client backed by the given Caller.
func NewClient(c Caller) *Client { return &Client{API: c} }

// Information calls get_server_information and decodes the response.
func (c *Client) Information(ctx context.Context) (ServiceList, error) {
	resp, err := c.API.Call(ctx, "get_server_information", nil)
	if err != nil {
		return nil, err
	}
	list, err := DecodeServices(resp.Body.ReturnInfo)
	if err != nil {
		return nil, fmt.Errorf("server: get_server_information: %w", err)
	}
	return list, nil
}

// DecodeServices maps ReturnInfo (an array of Maps) into the typed
// ServiceList.
func DecodeServices(returnInfo soap.Value) (ServiceList, error) {
	out, err := soap.DecodeArray(returnInfo, "server", func(item soap.Value) Service {
		return Service{
			Service:       item.MapString("service"),
			Version:       item.MapString("version"),
			VersionType:   item.MapString("version_type"),
			Interface:     item.MapString("interface"),
			FileExtension: item.MapString("file_extension"),
			Distribution:  item.MapString("distribution"),
		}
	})
	if err != nil {
		return nil, err
	}
	return ServiceList(out), nil
}

// TableHeaders returns the columns used by --output=table.
func (ServiceList) TableHeaders() []string {
	return []string{"SERVICE", "VERSION", "INTERFACE", "FILE_EXT", "DETAIL"}
}

// TableRows emits one row per service entry. The DETAIL column merges
// version_type and distribution so heterogeneous entries fit in one
// rectangular table.
func (l ServiceList) TableRows() [][]string {
	rows := make([][]string, 0, len(l))
	for _, s := range l {
		detail := s.VersionType
		if detail == "" {
			detail = s.Distribution
		}
		rows = append(rows, []string{
			s.Service,
			s.Version,
			s.Interface,
			s.FileExtension,
			detail,
		})
	}
	return rows
}
