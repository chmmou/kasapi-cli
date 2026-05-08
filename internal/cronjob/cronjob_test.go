package cronjob_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/cronjob"
	"github.com/chmmou/kasapi-cli/internal/soap"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("repo root not found from %q", file)
		}
		dir = parent
	}
}

func decodeFixture(t *testing.T, name string) *soap.Response {
	t.Helper()
	path := filepath.Join(repoRoot(t), "testdata", "cronjob", name)
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", name, err)
	}
	defer func() { _ = f.Close() }()
	resp, err := soap.Decode(f)
	if err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
	return resp
}

type fakeCaller struct {
	resp *soap.Response
	err  error

	gotAction string
	gotParams map[string]any
}

func (f *fakeCaller) Call(_ context.Context, action string, params map[string]any) (*soap.Response, error) {
	f.gotAction = action
	f.gotParams = params
	return f.resp, f.err
}

func TestDecodeCronjobs(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_cronjobs_response_success.xml")
	got, err := cronjob.DecodeCronjobs(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeCronjobs: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	first := got[0]
	if first.ID != "325206" {
		t.Errorf("ID = %q, want 325206", first.ID)
	}
	if first.Comment != "Hourly Cron" {
		t.Errorf("Comment = %q", first.Comment)
	}
	if first.Protocol != "https" {
		t.Errorf("Protocol = %q", first.Protocol)
	}
	if first.HTTPURL != "example.de/cron.php" {
		t.Errorf("HTTPURL = %q", first.HTTPURL)
	}
	if first.Schedule() != "59 * * * *" {
		t.Errorf("Schedule = %q, want '59 * * * *'", first.Schedule())
	}
	// xsi:nil for shell_command and timeout must round-trip cleanly.
	if first.ShellCommand != "" {
		t.Errorf("ShellCommand = %q, want empty (xsi:nil)", first.ShellCommand)
	}
	if first.Timeout != 0 {
		t.Errorf("Timeout = %d, want 0 (xsi:nil)", first.Timeout)
	}
	// The KAS API spells the address key with a single 'd'; verify
	// the wire spelling round-trips without renaming.
	if first.MailAdress != "cronjob@example.de" {
		t.Errorf("MailAdress = %q", first.MailAdress)
	}
	if first.IsActive != "N" {
		t.Errorf("IsActive = %q, want N", first.IsActive)
	}
	// Second entry uses hour="*/1" — confirms the schedule string is
	// preserved verbatim.
	if got[1].Hour != "*/1" {
		t.Errorf("got[1].Hour = %q, want */1", got[1].Hour)
	}
}

func TestDecodeCronjobSingular(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_cronjob_response_success.xml")
	got, err := cronjob.DecodeCronjobs(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeCronjobs: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	c := got[0]
	if c.ID != "325208" {
		t.Errorf("ID = %q, want 325208", c.ID)
	}
	if c.Schedule() != "59 */1 * * *" {
		t.Errorf("Schedule = %q", c.Schedule())
	}
}

func TestCronjobTarget(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		c    cronjob.Cronjob
		want string
	}{
		{
			name: "https",
			c:    cronjob.Cronjob{Protocol: "https", HTTPURL: "example.de/cron.php"},
			want: "https://example.de/cron.php",
		},
		{
			name: "http",
			c:    cronjob.Cronjob{Protocol: "http", HTTPURL: "example.de/cron.php"},
			want: "http://example.de/cron.php",
		},
		{
			name: "shell",
			c:    cronjob.Cronjob{Protocol: "shell", ShellCommand: "/usr/bin/php cron.php"},
			want: "/usr/bin/php cron.php",
		},
		{
			name: "https without url",
			c:    cronjob.Cronjob{Protocol: "https"},
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.c.Target(); got != tc.want {
				t.Errorf("Target() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestClientList(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_cronjobs_response_success.xml")
	fc := &fakeCaller{resp: resp}
	list, err := cronjob.NewClient(fc).List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if fc.gotAction != "get_cronjobs" {
		t.Errorf("action = %q, want get_cronjobs", fc.gotAction)
	}
	if fc.gotParams != nil {
		t.Errorf("params = %v, want nil", fc.gotParams)
	}
	if len(list) != 2 {
		t.Errorf("len = %d, want 2", len(list))
	}
}

func TestClientGet(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_cronjob_response_success.xml")
	fc := &fakeCaller{resp: resp}
	c, err := cronjob.NewClient(fc).Get(context.Background(), "325208")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if fc.gotAction != "get_cronjobs" {
		t.Errorf("action = %q, want get_cronjobs", fc.gotAction)
	}
	if got, _ := fc.gotParams["cronjob_id"].(string); got != "325208" {
		t.Errorf("params[cronjob_id] = %v, want 325208", fc.gotParams["cronjob_id"])
	}
	if c.ID != "325208" {
		t.Errorf("ID = %q, want 325208", c.ID)
	}
}

func TestClientGetEmptyID(t *testing.T) {
	t.Parallel()
	c := cronjob.NewClient(&fakeCaller{})
	if _, err := c.Get(context.Background(), ""); err == nil {
		t.Errorf("Get(\"\") err = nil, want validation error")
	}
}

func TestClientGetNotFound(t *testing.T) {
	t.Parallel()
	resp := &soap.Response{Body: soap.ResponseBody{ReturnInfo: soap.Value{Kind: soap.KindArray}}}
	c := cronjob.NewClient(&fakeCaller{resp: resp})
	if _, err := c.Get(context.Background(), "999999"); err == nil {
		t.Errorf("Get on empty result err = nil, want not-found")
	}
}

func TestClientPropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := cronjob.NewClient(&fakeCaller{err: want})
	if _, err := c.List(context.Background()); !errors.Is(err, want) {
		t.Errorf("List err = %v, want %v wrapped", err, want)
	}
	if _, err := c.Get(context.Background(), "325208"); !errors.Is(err, want) {
		t.Errorf("Get err = %v, want %v wrapped", err, want)
	}
}

func TestCronjobListTabular(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_cronjobs_response_success.xml")
	list, _ := cronjob.DecodeCronjobs(resp.Body.ReturnInfo)
	headers := list.TableHeaders()
	if headers[0] != "ID" {
		t.Errorf("headers[0] = %q, want ID", headers[0])
	}
	rows := list.TableRows()
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	if rows[0][0] != "325206" {
		t.Errorf("rows[0][0] = %q, want 325206", rows[0][0])
	}
	if rows[0][2] != "59 * * * *" {
		t.Errorf("rows[0][SCHEDULE] = %q, want '59 * * * *'", rows[0][2])
	}
	if rows[0][4] != "https://example.de/cron.php" {
		t.Errorf("rows[0][TARGET] = %q", rows[0][4])
	}
}

func TestCronjobTabular(t *testing.T) {
	t.Parallel()
	resp := decodeFixture(t, "get_cronjob_response_success.xml")
	list, _ := cronjob.DecodeCronjobs(resp.Body.ReturnInfo)
	if len(list) != 1 {
		t.Fatalf("len = %d, want 1", len(list))
	}
	c := list[0]
	headers := c.TableHeaders()
	if headers[0] != "FIELD" || headers[1] != "VALUE" {
		t.Errorf("headers = %v, want [FIELD VALUE]", headers)
	}
	rows := c.TableRows()
	if rows[0][0] != "cronjob_id" || rows[0][1] != "325208" {
		t.Errorf("rows[0] = %v, want [cronjob_id 325208]", rows[0])
	}
}
