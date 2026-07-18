package dns

import (
	"context"
	"fmt"
	"strconv"

	"github.com/chmmou/kasapi-cli/internal/kasread"
	"github.com/chmmou/kasapi-cli/internal/soap"
)

// Caller is the subset of *api.Client this package depends on. The
// indirection keeps tests free of network setup: a fake Caller can
// return a *soap.Response decoded from a fixture.
type Caller = kasread.Caller

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
	c Caller
}

// NewClient returns a Client backed by the given Caller.
func NewClient(c Caller) *Client { return &Client{c: c} }

// Settings calls get_dns_settings for the given zone host and decodes
// the response into a RecordList. zoneHost is required (the zone the
// records belong to, e.g. "example.com"). recordID is optional and
// narrows the result to the single resource record with that ID —
// leave it empty to list every record in the zone.
func (cl *Client) Settings(ctx context.Context, zoneHost, recordID string) (RecordList, error) {
	if zoneHost == "" {
		return nil, fmt.Errorf("dns: zone_host is required")
	}
	params := map[string]any{"zone_host": zoneHost}
	if recordID != "" {
		params["record_id"] = recordID
	}
	resp, err := cl.c.Call(ctx, "get_dns_settings", params)
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
	out, err := soap.DecodeArray(returnInfo, "dns", func(item soap.Value) Record {
		return Record{
			Zone:       item.MapString("record_zone"),
			Name:       item.MapString("record_name"),
			Type:       item.MapString("record_type"),
			Data:       item.MapString("record_data"),
			Aux:        item.MapInt("record_aux"),
			ID:         item.MapString("record_id"),
			Changeable: item.MapString("record_changeable"),
			Deleteable: item.MapString("record_deleteable"),
		}
	})
	if err != nil {
		return nil, err
	}
	return RecordList(out), nil
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
