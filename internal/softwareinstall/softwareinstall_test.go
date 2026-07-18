package softwareinstall_test

import (
	"context"
	"errors"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/softwareinstall"
	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestDecodeSoftwareInstalls(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "softwareinstall/get_softwareinstall_response_success_all.xml")
	got, err := softwareinstall.DecodeSoftwareInstalls(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeSoftwareInstalls: %v", err)
	}
	if len(got) != 22 {
		t.Fatalf("len = %d, want 22", len(got))
	}
	// Verify a few entries can be located by ID and that the
	// runtime-version fields are preserved verbatim, including the
	// 0.0 sentinel for "not applicable".
	byID := map[string]softwareinstall.SoftwareInstall{}
	for _, s := range got {
		byID[s.ID] = s
	}
	joomla, ok := byID["joomla_v6.1.0"]
	if !ok {
		t.Fatalf("joomla_v6.1.0 not in list")
	}
	if joomla.Name != "Joomla!" {
		t.Errorf("Name = %q, want Joomla!", joomla.Name)
	}
	if joomla.Category != "CMS" {
		t.Errorf("Category = %q", joomla.Category)
	}
	if joomla.PHPVersion != "8.4" {
		t.Errorf("PHPVersion = %q", joomla.PHPVersion)
	}
	if joomla.MySQLVersion != "0.0" {
		t.Errorf("MySQLVersion = %q, want 0.0 (sentinel preserved)", joomla.MySQLVersion)
	}
	if joomla.MariaDBVersion != "10.5" {
		t.Errorf("MariaDBVersion = %q", joomla.MariaDBVersion)
	}
	if joomla.CanBeInstalled != "Y" {
		t.Errorf("CanBeInstalled = %q", joomla.CanBeInstalled)
	}
}

func TestDecodeSoftwareInstallSingular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "softwareinstall/get_softwareinstall_response_success.xml")
	got, err := softwareinstall.DecodeSoftwareInstalls(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeSoftwareInstalls: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	s := got[0]
	if s.ID != "joomla_v6.1.0" {
		t.Errorf("ID = %q", s.ID)
	}
	if s.Description == "" {
		t.Error("Description not populated")
	}
	if s.Image == "" {
		t.Error("Image not populated (expected base64 data URI)")
	}
}

func TestClientList(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "softwareinstall/get_softwareinstall_response_success_all.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	list, err := softwareinstall.NewClient(fc).List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if fc.GotAction != "get_softwareinstall" {
		t.Errorf("action = %q, want get_softwareinstall (singular for both)", fc.GotAction)
	}
	if fc.GotParams != nil {
		t.Errorf("params = %v, want nil", fc.GotParams)
	}
	if len(list) == 0 {
		t.Errorf("len = %d, want a non-empty list", len(list))
	}
}

func TestClientGet(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "softwareinstall/get_softwareinstall_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	s, err := softwareinstall.NewClient(fc).Get(context.Background(), "joomla_v6.1.0")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if fc.GotAction != "get_softwareinstall" {
		t.Errorf("action = %q", fc.GotAction)
	}
	if got, _ := fc.GotParams["software_id"].(string); got != "joomla_v6.1.0" {
		t.Errorf("params[software_id] = %v", fc.GotParams["software_id"])
	}
	if s.ID != "joomla_v6.1.0" {
		t.Errorf("ID = %q", s.ID)
	}
}

func TestClientGetEmptyID(t *testing.T) {
	t.Parallel()
	c := softwareinstall.NewClient(&testutil.FakeCaller{})
	if _, err := c.Get(context.Background(), ""); err == nil {
		t.Errorf("Get(\"\") err = nil, want validation error")
	}
}

func TestClientGetNotFound(t *testing.T) {
	t.Parallel()
	resp := &soap.Response{Body: soap.ResponseBody{ReturnInfo: soap.Value{Kind: soap.KindArray}}}
	c := softwareinstall.NewClient(&testutil.FakeCaller{Resp: resp})
	if _, err := c.Get(context.Background(), "nope_v0.0"); err == nil {
		t.Errorf("Get on empty result err = nil, want not-found")
	}
}

func TestClientPropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := softwareinstall.NewClient(&testutil.FakeCaller{Err: want})
	if _, err := c.List(context.Background()); !errors.Is(err, want) {
		t.Errorf("List err = %v, want %v wrapped", err, want)
	}
	if _, err := c.Get(context.Background(), "joomla_v6.1.0"); !errors.Is(err, want) {
		t.Errorf("Get err = %v, want %v wrapped", err, want)
	}
}

func TestSoftwareInstallListTabular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "softwareinstall/get_softwareinstall_response_success_all.xml")
	list, _ := softwareinstall.DecodeSoftwareInstalls(resp.Body.ReturnInfo)
	headers := list.TableHeaders()
	if headers[0] != "ID" {
		t.Errorf("headers[0] = %q, want ID", headers[0])
	}
	rows := list.TableRows()
	if len(rows) != 22 {
		t.Fatalf("rows = %d, want 22", len(rows))
	}
	// PHP / DB columns must collapse the {from, upto} pair.
	for _, row := range rows {
		if row[0] == "joomla_v6.1.0" {
			if row[4] != "8.4" {
				t.Errorf("PHP column for joomla = %q, want 8.4 (from==upto collapses)", row[4])
			}
			if row[5] != "MariaDB 10.5..12.0" {
				t.Errorf("DB column for joomla = %q, want 'MariaDB 10.5..12.0'", row[5])
			}
		}
	}
}

func TestSoftwareInstallTabular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "softwareinstall/get_softwareinstall_response_success.xml")
	list, _ := softwareinstall.DecodeSoftwareInstalls(resp.Body.ReturnInfo)
	if len(list) != 1 {
		t.Fatalf("len = %d, want 1", len(list))
	}
	s := list[0]
	headers := s.TableHeaders()
	if headers[0] != "FIELD" || headers[1] != "VALUE" {
		t.Errorf("headers = %v, want [FIELD VALUE]", headers)
	}
	rows := s.TableRows()
	if rows[0][0] != "software_id" || rows[0][1] != "joomla_v6.1.0" {
		t.Errorf("rows[0] = %v, want [software_id joomla_v6.1.0]", rows[0])
	}
	// image must NOT appear in the K/V table.
	for _, row := range rows {
		if row[0] == "image" {
			t.Errorf("image leaked into table view: %v", row)
		}
	}
}
