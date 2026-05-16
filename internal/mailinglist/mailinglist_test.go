package mailinglist_test

import (
	"context"
	"errors"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/mailinglist"
	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestDecodeMailingLists(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "mailinglist/get_mailinglists_response_success.xml")
	got, err := mailinglist.DecodeMailingLists(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeMailingLists: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (per fixture arrayType)", len(got))
	}
	m := got[0]
	if m.Name != "announce@example.com" {
		t.Errorf("Name = %q", m.Name)
	}
	if m.Admin != "admin@example.com" {
		t.Errorf("Admin = %q", m.Admin)
	}
	if m.URL == "" {
		t.Errorf("URL empty")
	}
}

func TestDecodeMailingListSingular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "mailinglist/get_mailinglist_response_success.xml")
	got, err := mailinglist.DecodeMailingLists(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeMailingLists: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Name != "announce@example.com" {
		t.Errorf("Name = %q", got[0].Name)
	}
}

func TestClientList(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "mailinglist/get_mailinglists_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	list, err := mailinglist.NewClient(fc).List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if fc.GotAction != "get_mailinglists" {
		t.Errorf("action = %q, want get_mailinglists", fc.GotAction)
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
	resp := testutil.DecodeFixture(t, "mailinglist/get_mailinglist_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	m, err := mailinglist.NewClient(fc).Get(context.Background(), "announce@example.com")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if fc.GotAction != "get_mailinglists" {
		t.Errorf("action = %q, want get_mailinglists", fc.GotAction)
	}
	if got, _ := fc.GotParams["mailinglist_name"].(string); got != "announce@example.com" {
		t.Errorf("params[mailinglist_name] = %v, want announce@example.com", fc.GotParams["mailinglist_name"])
	}
	if m.Name != "announce@example.com" {
		t.Errorf("Name = %q", m.Name)
	}
}

func TestClientGetEmptyName(t *testing.T) {
	t.Parallel()
	c := mailinglist.NewClient(&testutil.FakeCaller{})
	if _, err := c.Get(context.Background(), ""); err == nil {
		t.Errorf("Get(\"\") err = nil, want validation error")
	}
}

func TestClientGetNotFound(t *testing.T) {
	t.Parallel()
	resp := &soap.Response{Body: soap.ResponseBody{ReturnInfo: soap.Value{Kind: soap.KindArray}}}
	c := mailinglist.NewClient(&testutil.FakeCaller{Resp: resp})
	if _, err := c.Get(context.Background(), "missing@example.com"); err == nil {
		t.Errorf("Get on empty result err = nil, want not-found")
	}
}

func TestClientPropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := mailinglist.NewClient(&testutil.FakeCaller{Err: want})
	if _, err := c.List(context.Background()); !errors.Is(err, want) {
		t.Errorf("List err = %v, want %v wrapped", err, want)
	}
	if _, err := c.Get(context.Background(), "announce@example.com"); !errors.Is(err, want) {
		t.Errorf("Get err = %v, want %v wrapped", err, want)
	}
}

func TestMailingListListTabular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "mailinglist/get_mailinglists_response_success.xml")
	list, _ := mailinglist.DecodeMailingLists(resp.Body.ReturnInfo)
	rows := list.TableRows()
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	if rows[0][0] != "announce@example.com" {
		t.Errorf("rows[0][0] = %q", rows[0][0])
	}
}

func TestMailingListSingularTabular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "mailinglist/get_mailinglist_response_success.xml")
	list, _ := mailinglist.DecodeMailingLists(resp.Body.ReturnInfo)
	if len(list) != 1 {
		t.Fatalf("len = %d, want 1", len(list))
	}
	// The order is part of the user-visible table contract: identity
	// (name, admin, url) before lifecycle state (in_progress). Pin it
	// by index so a refactor reordering the slice does not slip
	// through silently.
	want := [][]string{
		{"mailinglist_name", "announce@example.com"},
		{"mailinglist_admin", "admin@example.com"},
		{"mailinglist_url", "https://lists.example.com/mailman/listinfo/announce"},
		{"in_progress", "FALSE"},
	}
	rows := list[0].TableRows()
	if len(rows) != len(want) {
		t.Fatalf("rows = %d, want %d", len(rows), len(want))
	}
	for i, r := range rows {
		if r[0] != want[i][0] || r[1] != want[i][1] {
			t.Errorf("row[%d] = %v, want %v", i, r, want[i])
		}
	}
	hdr := (mailinglist.MailingList{}).TableHeaders()
	if len(hdr) != 2 || hdr[0] != "FIELD" || hdr[1] != "VALUE" {
		t.Errorf("headers = %v, want [FIELD VALUE]", hdr)
	}
}
