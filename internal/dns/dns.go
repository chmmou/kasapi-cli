package dns

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

// Record is one DNS resource record returned by get_dns_settings.
// record_id is xsd:string in the wire format and we keep it as a
// string here — the value identifies the record on update / delete
// calls but is otherwise opaque.
type Record struct {
	Zone       string `json:"record_zone" yaml:"record_zone"`
	Name       string `json:"record_name" yaml:"record_name"`
	Type       string `json:"record_type" yaml:"record_type"`
	Data       string `json:"record_data" yaml:"record_data"`
	Aux        int    `json:"record_aux" yaml:"record_aux"`
	ID         string `json:"record_id" yaml:"record_id"`
	Changeable string `json:"record_changeable" yaml:"record_changeable"`
	Deleteable string `json:"record_deleteable" yaml:"record_deleteable"`
}

// RecordList is the typed payload of get_dns_settings; satisfies
// cli.Tabular.
type RecordList []Record

// Client groups the read endpoints scoped to DNS settings.
type Client struct {
	API Caller
}

// NewClient returns a Client backed by the given Caller.
func NewClient(c Caller) *Client { return &Client{API: c} }

// Settings calls get_dns_settings for the given zone host and decodes
// the response into a RecordList. zoneHost is required (the zone the
// records belong to, e.g. "example.com"). nameserver is optional and
// pinpoints which authoritative NS to query when the zone is served
// by more than one — leave it empty to use the KAS default.
func (c *Client) Settings(ctx context.Context, zoneHost, nameserver string) (RecordList, error) {
	if zoneHost == "" {
		return nil, fmt.Errorf("dns: zone_host is required")
	}
	params := map[string]any{"zone_host": zoneHost}
	if nameserver != "" {
		params["nameserver"] = nameserver
	}
	resp, err := c.API.Call(ctx, "get_dns_settings", params)
	if err != nil {
		return nil, err
	}
	list, err := DecodeRecords(resp.Body.ReturnInfo)
	if err != nil {
		return nil, fmt.Errorf("dns: get_dns_settings: %w", err)
	}
	return list, nil
}

// DecodeRecords maps the ReturnInfo of a get_dns_settings response
// (an Array of Maps) into the typed RecordList.
func DecodeRecords(returnInfo soap.Value) (RecordList, error) {
	if returnInfo.Kind != soap.KindArray {
		return nil, fmt.Errorf("dns: expected ReturnInfo array, got kind %d", returnInfo.Kind)
	}
	out := make(RecordList, 0, len(returnInfo.Array))
	for i, item := range returnInfo.Array {
		if item.Kind != soap.KindMap {
			return nil, fmt.Errorf("dns: ReturnInfo[%d] is not a Map", i)
		}
		out = append(out, Record{
			Zone:       getString(item, "record_zone"),
			Name:       getString(item, "record_name"),
			Type:       getString(item, "record_type"),
			Data:       getString(item, "record_data"),
			Aux:        getInt(item, "record_aux"),
			ID:         getString(item, "record_id"),
			Changeable: getString(item, "record_changeable"),
			Deleteable: getString(item, "record_deleteable"),
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
// RecordList.
func (RecordList) TableHeaders() []string {
	return []string{"ID", "ZONE", "NAME", "TYPE", "AUX", "DATA", "CHG", "DEL"}
}

// TableRows emits one row per Record entry.
func (l RecordList) TableRows() [][]string {
	rows := make([][]string, 0, len(l))
	for _, r := range l {
		rows = append(rows, []string{
			r.ID,
			r.Zone,
			r.Name,
			r.Type,
			strconv.Itoa(r.Aux),
			r.Data,
			r.Changeable,
			r.Deleteable,
		})
	}
	return rows
}
