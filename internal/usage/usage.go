package usage

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

// Space is one entry of get_space. Each entry covers an account-login
// (the main account and each sub-account) with byte counts split by
// resource type. KAS reports webspace and max_webspace as xsd:int and
// the per-resource breakdowns as xsd:string-encoded numbers; we parse
// both into int64 so the per-account totals add up without overflow.
//
// UsedWebspace is the sum of UsedHTDocsSpace, UsedChrootSpace,
// UsedDatabaseSpace, and UsedMailaccountSpace — do not add the
// sub-buckets to UsedWebspace when computing totals.
type Space struct {
	AccountLogin         string `json:"account_login" yaml:"account_login"`
	LastCalculation      int64  `json:"last_calculation" yaml:"last_calculation"`
	UsedHTDocsSpace      int64  `json:"used_htdocs_space" yaml:"used_htdocs_space"`
	UsedChrootSpace      int64  `json:"used_chroot_space" yaml:"used_chroot_space"`
	UsedDatabaseSpace    int64  `json:"used_database_space" yaml:"used_database_space"`
	UsedMailaccountSpace int64  `json:"used_mailaccount_space" yaml:"used_mailaccount_space"`
	UsedWebspace         int64  `json:"used_webspace" yaml:"used_webspace"`
	MaxWebspace          int64  `json:"max_webspace" yaml:"max_webspace"`
}

// SpaceList is the typed payload of get_space; satisfies cli.Tabular.
type SpaceList []Space

// SpaceUsage is one entry of get_space_usage: a directory together with
// its file count and aggregate byte size. has_sub_dirs signals whether
// the entry can be drilled into with another get_space_usage call.
type SpaceUsage struct {
	Directory       string `json:"directory" yaml:"directory"`
	Count           int64  `json:"count" yaml:"count"`
	Bytes           int64  `json:"bytes" yaml:"bytes"`
	LastCalculation int64  `json:"last_calculation" yaml:"last_calculation"`
	HasSubDirs      bool   `json:"has_sub_dirs" yaml:"has_sub_dirs"`
}

// SpaceUsageList is the typed payload of get_space_usage; satisfies
// cli.Tabular.
type SpaceUsageList []SpaceUsage

// Traffic is one entry of get_traffic. KAS interleaves a monthly
// summary entry (Day == 0, AccountLogin set, Comment populated) with
// per-day rows (Day 1..31). FTP traffic / hits are returned as xsi:nil
// when no data is available; we map nil to zero. Unknown bytes counts
// thus look identical to "no traffic"; use the Comment field on the
// summary row to disambiguate when the distinction matters.
type Traffic struct {
	AccountLogin string `json:"account_login,omitempty" yaml:"account_login,omitempty"`
	Year         int    `json:"year" yaml:"year"`
	Month        int    `json:"month" yaml:"month"`
	Day          int    `json:"day,omitempty" yaml:"day,omitempty"`
	HTTPTraffic  int64  `json:"http_traffic" yaml:"http_traffic"`
	FTPTraffic   int64  `json:"ftp_traffic" yaml:"ftp_traffic"`
	HTTPHits     int64  `json:"http_hits" yaml:"http_hits"`
	FTPHits      int64  `json:"ftp_hits" yaml:"ftp_hits"`
	Comment      string `json:"comment,omitempty" yaml:"comment,omitempty"`
}

// IsSummary reports whether t is the monthly summary row that KAS
// emits alongside the per-day entries. The summary is identified by a
// zero Day; literal day-zero never occurs in get_traffic responses.
func (t Traffic) IsSummary() bool { return t.Day == 0 }

// TrafficList is the typed payload of get_traffic; satisfies
// cli.Tabular.
type TrafficList []Traffic

// Client groups the read endpoints scoped to webspace + traffic
// counters: get_space, get_space_usage, get_traffic.
type Client struct {
	API Caller
}

// NewClient returns a Client backed by the given Caller.
func NewClient(c Caller) *Client { return &Client{API: c} }

// Space calls get_space and decodes the response into SpaceList.
//
// The KAS API accepts optional show_subaccounts and show_details
// parameters; both default to "Y" server-side. We omit them — callers
// that need a different scope can call (*api.Client).Call directly.
func (c *Client) Space(ctx context.Context) (SpaceList, error) {
	resp, err := c.API.Call(ctx, "get_space", nil)
	if err != nil {
		return nil, err
	}
	list, err := DecodeSpace(resp.Body.ReturnInfo)
	if err != nil {
		return nil, fmt.Errorf("usage: get_space: %w", err)
	}
	return list, nil
}

// SpaceUsage calls get_space_usage for the given directory and decodes
// the response. An empty directory queries the document-root level.
func (c *Client) SpaceUsage(ctx context.Context, directory string) (SpaceUsageList, error) {
	var params map[string]any
	if directory != "" {
		params = map[string]any{"directory": directory}
	}
	resp, err := c.API.Call(ctx, "get_space_usage", params)
	if err != nil {
		return nil, err
	}
	list, err := DecodeSpaceUsage(resp.Body.ReturnInfo)
	if err != nil {
		return nil, fmt.Errorf("usage: get_space_usage: %w", err)
	}
	return list, nil
}

// Traffic calls get_traffic and decodes the response. year and month
// are optional; pass 0/0 to query the current month. month is encoded
// as a zero-padded string ("01".."12") because KAS rejects "1" with a
// syntax error.
func (c *Client) Traffic(ctx context.Context, year, month int) (TrafficList, error) {
	var params map[string]any
	if year != 0 || month != 0 {
		params = make(map[string]any, 2)
		if year != 0 {
			params["year"] = strconv.Itoa(year)
		}
		if month != 0 {
			params["month"] = fmt.Sprintf("%02d", month)
		}
	}
	resp, err := c.API.Call(ctx, "get_traffic", params)
	if err != nil {
		return nil, err
	}
	list, err := DecodeTraffic(resp.Body.ReturnInfo)
	if err != nil {
		return nil, fmt.Errorf("usage: get_traffic: %w", err)
	}
	return list, nil
}

// DecodeSpace maps the ReturnInfo of a get_space response (an Array of
// Maps) into the typed SpaceList.
func DecodeSpace(returnInfo soap.Value) (SpaceList, error) {
	out, err := soap.DecodeArray(returnInfo, "usage", func(item soap.Value) Space {
		return Space{
			AccountLogin:         item.MapString("account_login"),
			LastCalculation:      item.MapInt64("last_calculation"),
			UsedHTDocsSpace:      item.MapInt64("used_htdocs_space"),
			UsedChrootSpace:      item.MapInt64("used_chroot_space"),
			UsedDatabaseSpace:    item.MapInt64("used_database_space"),
			UsedMailaccountSpace: item.MapInt64("used_mailaccount_space"),
			UsedWebspace:         item.MapInt64("used_webspace"),
			MaxWebspace:          item.MapInt64("max_webspace"),
		}
	})
	if err != nil {
		return nil, err
	}
	return SpaceList(out), nil
}

// DecodeSpaceUsage maps the ReturnInfo of a get_space_usage response
// (an Array of Maps) into the typed SpaceUsageList.
func DecodeSpaceUsage(returnInfo soap.Value) (SpaceUsageList, error) {
	out, err := soap.DecodeArray(returnInfo, "usage", func(item soap.Value) SpaceUsage {
		return SpaceUsage{
			Directory:       item.MapString("directory"),
			Count:           item.MapInt64("count"),
			Bytes:           item.MapInt64("bytes"),
			LastCalculation: item.MapInt64("last_calculation"),
			HasSubDirs:      getYN(item, "has_sub_dirs"),
		}
	})
	if err != nil {
		return nil, err
	}
	return SpaceUsageList(out), nil
}

// DecodeTraffic maps the ReturnInfo of a get_traffic response (a Map of
// Maps keyed by "0" for the monthly summary and "01".."31" for daily
// rows) into the typed TrafficList. The summary entry comes first; the
// remaining entries follow in the order KAS returned them.
func DecodeTraffic(returnInfo soap.Value) (TrafficList, error) {
	if returnInfo.Kind != soap.KindMap {
		return nil, fmt.Errorf("usage: expected ReturnInfo map, got kind %d", returnInfo.Kind)
	}
	out := make(TrafficList, 0, len(returnInfo.Map))
	for _, kv := range returnInfo.Map {
		if kv.Value.Kind != soap.KindMap {
			return nil, fmt.Errorf("usage: ReturnInfo entry %q is not a Map", kv.Key)
		}
		// year and month identify the period every traffic row (the
		// summary and each daily entry) belongs to and are present in
		// every captured row: a missing or malformed value is a corrupt
		// response, not a zero, so they are decoded strictly. day stays
		// lenient on purpose — the monthly summary row (key "0")
		// legitimately omits it. http_/ftp_traffic and *_hits also stay
		// lenient: KAS returns xsi:nil for a bucket with no data (the
		// fixture's ftp_* are nil), so a strict reading would turn a
		// real no-traffic response into a hard error.
		year, err := kv.Value.MapIntStrict("year")
		if err != nil {
			return nil, fmt.Errorf("usage: traffic entry %q: year: %w", kv.Key, err)
		}
		month, err := kv.Value.MapIntStrict("month")
		if err != nil {
			return nil, fmt.Errorf("usage: traffic entry %q: month: %w", kv.Key, err)
		}
		out = append(out, Traffic{
			AccountLogin: kv.Value.MapString("account_login"),
			Year:         year,
			Month:        month,
			Day:          kv.Value.MapInt("day"),
			HTTPTraffic:  kv.Value.MapInt64("http_traffic"),
			FTPTraffic:   kv.Value.MapInt64("ftp_traffic"),
			HTTPHits:     kv.Value.MapInt64("http_hits"),
			FTPHits:      kv.Value.MapInt64("ftp_hits"),
			Comment:      kv.Value.MapString("comment"),
		})
	}
	return out, nil
}

func getYN(m soap.Value, key string) bool {
	v, ok := m.Get(key)
	if !ok {
		return false
	}
	switch strings.ToUpper(strings.TrimSpace(v.AsString())) {
	case "Y", "YES", "TRUE", "1":
		return true
	}
	return false
}

// TableHeaders returns the columns used by --output=table for SpaceList.
func (SpaceList) TableHeaders() []string {
	return []string{"ACCOUNT", "USED", "MAX", "%", "HTDOCS", "DB", "MAIL"}
}

// TableRows emits one row per Space entry. USED/MAX are kept as raw
// byte counts so the output stays unit-agnostic; the % column is the
// usage ratio rounded to one decimal so an at-a-glance read is easy.
func (l SpaceList) TableRows() [][]string {
	rows := make([][]string, 0, len(l))
	for _, s := range l {
		ratio := ""
		if s.MaxWebspace > 0 {
			ratio = strconv.FormatFloat(float64(s.UsedWebspace)*100/float64(s.MaxWebspace), 'f', 1, 64)
		}
		rows = append(rows, []string{
			s.AccountLogin,
			strconv.FormatInt(s.UsedWebspace, 10),
			strconv.FormatInt(s.MaxWebspace, 10),
			ratio,
			strconv.FormatInt(s.UsedHTDocsSpace, 10),
			strconv.FormatInt(s.UsedDatabaseSpace, 10),
			strconv.FormatInt(s.UsedMailaccountSpace, 10),
		})
	}
	return rows
}

// TableHeaders returns the columns used by --output=table for
// SpaceUsageList.
func (SpaceUsageList) TableHeaders() []string {
	return []string{"DIRECTORY", "COUNT", "BYTES", "SUBDIRS"}
}

// TableRows emits one row per SpaceUsage entry.
func (l SpaceUsageList) TableRows() [][]string {
	rows := make([][]string, 0, len(l))
	for _, u := range l {
		sub := "N"
		if u.HasSubDirs {
			sub = "Y"
		}
		rows = append(rows, []string{
			u.Directory,
			strconv.FormatInt(u.Count, 10),
			strconv.FormatInt(u.Bytes, 10),
			sub,
		})
	}
	return rows
}

// TableHeaders returns the columns used by --output=table for
// TrafficList. The summary row uses "*" in the DAY column.
func (TrafficList) TableHeaders() []string {
	return []string{"ACCOUNT", "YEAR", "MONTH", "DAY", "HTTP_BYTES", "FTP_BYTES", "HTTP_HITS", "FTP_HITS"}
}

// TableRows emits one row per Traffic entry.
func (l TrafficList) TableRows() [][]string {
	rows := make([][]string, 0, len(l))
	for _, t := range l {
		day := "*"
		if !t.IsSummary() {
			day = fmt.Sprintf("%02d", t.Day)
		}
		rows = append(rows, []string{
			t.AccountLogin,
			strconv.Itoa(t.Year),
			fmt.Sprintf("%02d", t.Month),
			day,
			strconv.FormatInt(t.HTTPTraffic, 10),
			strconv.FormatInt(t.FTPTraffic, 10),
			strconv.FormatInt(t.HTTPHits, 10),
			strconv.FormatInt(t.FTPHits, 10),
		})
	}
	return rows
}
