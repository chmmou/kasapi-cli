package cronjob

import (
	"context"
	"strconv"
	"strings"

	"github.com/chmmou/kasapi-cli/internal/kasread"
	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/tablefmt"
)

// Caller is the subset of *api.Client this package depends on. The
// indirection keeps tests free of network setup: a fake Caller can
// return a *soap.Response decoded from a fixture.
type Caller = kasread.Caller

// Cronjob is one entry of get_cronjobs. The list and singular views
// (the latter being get_cronjobs called with a cronjob_id filter)
// return the same Map shape, so a single struct covers both.
//
// shell_command and timeout are returned as xsi:nil for cronjobs that
// were configured with the HTTP target instead — they decode to the
// zero value here and are flagged with omitempty so a JSON/YAML
// round-trip does not invent values that were never set.
type Cronjob struct {
	ID      string `json:"cronjob_id" yaml:"cronjob_id"`
	Comment string `json:"cronjob_comment" yaml:"cronjob_comment"`

	ShellCommand string `json:"shell_command,omitempty" yaml:"shell_command,omitempty"`
	Timeout      int    `json:"timeout,omitempty" yaml:"timeout,omitempty"`

	Protocol     string `json:"protocol" yaml:"protocol"`
	HTTPURL      string `json:"http_url" yaml:"http_url"`
	HTTPUser     string `json:"http_user" yaml:"http_user"`
	HTTPPassword string `json:"http_password,omitempty" yaml:"http_password,omitempty"`

	// Schedule fields are returned verbatim as cron expression
	// fragments ("*", "*/5", "1,15,30", …) so we keep them as
	// strings rather than parsing them.
	Minute     string `json:"minute" yaml:"minute"`
	Hour       string `json:"hour" yaml:"hour"`
	DayOfMonth string `json:"day_of_month" yaml:"day_of_month"`
	Month      string `json:"month" yaml:"month"`
	DayOfWeek  string `json:"day_of_week" yaml:"day_of_week"`

	// The KAS API spells the address key with a single 'd'
	// (mail_adress). The struct mirrors the wire key verbatim.
	MailAdress    string `json:"mail_adress" yaml:"mail_adress"`
	MailCondition string `json:"mail_condition" yaml:"mail_condition"`
	MailSubject   string `json:"mail_subject" yaml:"mail_subject"`

	IsActive string `json:"is_active" yaml:"is_active"`
}

// Schedule returns the five cron fields joined with single spaces in
// their canonical order ("min hour dom month dow"), matching the
// crontab(5) layout.
func (c Cronjob) Schedule() string {
	return strings.Join([]string{c.Minute, c.Hour, c.DayOfMonth, c.Month, c.DayOfWeek}, " ")
}

// Target returns the cronjob's primary trigger target — the configured
// http(s) URL when the protocol is http/https, the shell command
// otherwise. It is used by the table view as a single "what runs"
// column instead of leaking both fields side-by-side.
func (c Cronjob) Target() string {
	switch c.Protocol {
	case "http", "https":
		if c.HTTPURL == "" {
			return ""
		}
		return c.Protocol + "://" + c.HTTPURL
	default:
		return c.ShellCommand
	}
}

// CronjobList is the typed payload of get_cronjobs; satisfies
// cli.Tabular.
type CronjobList []Cronjob

// Client groups the read endpoint scoped to cronjobs (get_cronjobs,
// list and singular) and the write endpoints add_cronjob /
// update_cronjob / delete_cronjob (see write.go). The raw Caller is
// kept alongside the read helper so the write methods can dispatch
// their own KAS actions through the shared kaswrite seam.
type Client struct {
	lg kasread.ListGet[CronjobList, Cronjob]
	c  Caller
}

// NewClient returns a Client backed by the given Caller.
func NewClient(c Caller) *Client {
	return &Client{
		lg: kasread.ListGet[CronjobList, Cronjob]{
			Caller:    c,
			Action:    "get_cronjobs",
			Label:     "cronjob",
			ArgName:   "id",
			FilterKey: FieldID,
			Decoder:   DecodeCronjobs,
		},
		c: c,
	}
}

// List calls get_cronjobs without parameters and decodes the response
// into a CronjobList covering every cronjob visible to the login.
func (c *Client) List(ctx context.Context) (CronjobList, error) { return c.lg.List(ctx) }

// Get calls get_cronjobs with a cronjob_id filter and returns the
// single matching Cronjob. The KAS API still wraps the result in an
// array; we unwrap it here so callers do not have to. An empty array
// surfaces as a not-found error.
func (c *Client) Get(ctx context.Context, id string) (Cronjob, error) {
	return c.lg.Get(ctx, id)
}

// DecodeCronjobs maps the ReturnInfo of a get_cronjobs response (an
// Array of Maps) into the typed CronjobList.
func DecodeCronjobs(returnInfo soap.Value) (CronjobList, error) {
	out, err := soap.DecodeArray(returnInfo, "cronjob", func(item soap.Value) Cronjob {
		return Cronjob{
			ID:            item.MapString("cronjob_id"),
			Comment:       item.MapString("cronjob_comment"),
			ShellCommand:  item.MapString("shell_command"),
			Timeout:       item.MapInt("timeout"),
			Protocol:      item.MapString("protocol"),
			HTTPURL:       item.MapString("http_url"),
			HTTPUser:      item.MapString("http_user"),
			HTTPPassword:  item.MapString("http_password"),
			Minute:        item.MapString("minute"),
			Hour:          item.MapString("hour"),
			DayOfMonth:    item.MapString("day_of_month"),
			Month:         item.MapString("month"),
			DayOfWeek:     item.MapString("day_of_week"),
			MailAdress:    item.MapString("mail_adress"),
			MailCondition: item.MapString("mail_condition"),
			MailSubject:   item.MapString("mail_subject"),
			IsActive:      item.MapString("is_active"),
		}
	})
	if err != nil {
		return nil, err
	}
	return CronjobList(out), nil
}

// TableHeaders returns the columns used by --output=table for
// CronjobList.
func (CronjobList) TableHeaders() []string {
	return []string{"ID", "COMMENT", "SCHEDULE", "PROTOCOL", "TARGET", "ACTIVE"}
}

// TableRows emits one row per Cronjob entry. The schedule is rendered
// as a single crontab(5)-style string and the target collapses
// http_url and shell_command into one column so the table fits in a
// terminal.
func (l CronjobList) TableRows() [][]string {
	rows := make([][]string, 0, len(l))
	for _, c := range l {
		rows = append(rows, []string{
			c.ID,
			c.Comment,
			c.Schedule(),
			c.Protocol,
			c.Target(),
			c.IsActive,
		})
	}
	return rows
}

// TableHeaders for the singular Cronjob view: a key/value layout.
func (Cronjob) TableHeaders() []string {
	return tablefmt.FieldValueHeaders
}

// TableRows emits the scalar fields. http_password is intentionally
// omitted — consumers that need it should use --output=json|yaml.
func (c Cronjob) TableRows() [][]string {
	return [][]string{
		{"cronjob_id", c.ID},
		{"cronjob_comment", c.Comment},
		{"is_active", c.IsActive},
		{"protocol", c.Protocol},
		{"http_url", c.HTTPURL},
		{"http_user", c.HTTPUser},
		{"shell_command", c.ShellCommand},
		{"timeout", strconv.Itoa(c.Timeout)},
		{"schedule", c.Schedule()},
		{"minute", c.Minute},
		{"hour", c.Hour},
		{"day_of_month", c.DayOfMonth},
		{"month", c.Month},
		{"day_of_week", c.DayOfWeek},
		{"mail_adress", c.MailAdress},
		{"mail_condition", c.MailCondition},
		{"mail_subject", c.MailSubject},
	}
}
