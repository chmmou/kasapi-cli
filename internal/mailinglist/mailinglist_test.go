package mailinglist_test

import (
	"context"
	"errors"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/mailinglist"
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
	if m.Name != "list-example-org" {
		t.Errorf("Name = %q, want list-example-org", m.Name)
	}
	if m.Domain != "example.org" {
		t.Errorf("Domain = %q, want example.org", m.Domain)
	}
	if m.IsActive != "Y" {
		t.Errorf("IsActive = %q, want Y", m.IsActive)
	}
	if m.InProgress != "FALSE" {
		t.Errorf("InProgress = %q, want FALSE", m.InProgress)
	}
	// The list view does not return the singular-only fields.
	if m.Subscriber != "" || m.Config != "" || m.RestrictPost != "" {
		t.Errorf("singular-only fields populated in list view: %+v", m)
	}
	if got[1].InProgress != "TRUE" {
		t.Errorf("got[1].InProgress = %q, want TRUE", got[1].InProgress)
	}
}

func TestDecodeMailingListSingular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "mailinglist/get_mailinglists_response_success_single.xml")
	got, err := mailinglist.DecodeMailingLists(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeMailingLists: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	m := got[0]
	if m.Name != "announce-example-com" {
		t.Errorf("Name = %q, want announce-example-com", m.Name)
	}
	if m.Domain != "example.org" {
		t.Errorf("Domain = %q, want example.org", m.Domain)
	}
	if m.Config != "# Mailinglist configuration" {
		t.Errorf("Config = %q", m.Config)
	}
	if m.Subscriber != "" || m.RestrictPost != "" {
		t.Errorf("Subscriber/RestrictPost = %q/%q, want empty", m.Subscriber, m.RestrictPost)
	}
	if m.InProgress != "FALSE" {
		t.Errorf("InProgress = %q, want FALSE", m.InProgress)
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
	if len(list) != 2 {
		t.Errorf("len = %d, want 2", len(list))
	}
}

func TestClientListEmpty(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "mailinglist/get_mailinglists_response_success_empty.xml")
	list, err := mailinglist.NewClient(&testutil.FakeCaller{Resp: resp}).List(context.Background())
	if err != nil {
		t.Fatalf("List on empty result: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("len = %d, want 0", len(list))
	}
}

func TestClientGet(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "mailinglist/get_mailinglists_response_success_single.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	m, err := mailinglist.NewClient(fc).Get(context.Background(), "announce-example-com")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if fc.GotAction != "get_mailinglists" {
		t.Errorf("action = %q, want get_mailinglists", fc.GotAction)
	}
	if got, _ := fc.GotParams["mailinglist_name"].(string); got != "announce-example-com" {
		t.Errorf("params[mailinglist_name] = %v, want announce-example-com", fc.GotParams["mailinglist_name"])
	}
	if m.Name != "announce-example-com" {
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
	resp := testutil.DecodeFixture(t, "mailinglist/get_mailinglists_response_success_empty.xml")
	c := mailinglist.NewClient(&testutil.FakeCaller{Resp: resp})
	if _, err := c.Get(context.Background(), "list-not-exists-example-org"); err == nil {
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
	if _, err := c.Get(context.Background(), "announce-example-com"); !errors.Is(err, want) {
		t.Errorf("Get err = %v, want %v wrapped", err, want)
	}
}

func TestMailingListListTabular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "mailinglist/get_mailinglists_response_success.xml")
	list, _ := mailinglist.DecodeMailingLists(resp.Body.ReturnInfo)
	hdr := mailinglist.MailingListList(nil).TableHeaders()
	want := []string{"NAME", "DOMAIN", "ACTIVE", "IN_PROGRESS"}
	if len(hdr) != len(want) {
		t.Fatalf("headers = %v, want %v", hdr, want)
	}
	for i := range want {
		if hdr[i] != want[i] {
			t.Errorf("header[%d] = %q, want %q", i, hdr[i], want[i])
		}
	}
	rows := list.TableRows()
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	if rows[0][0] != "list-example-org" || rows[0][1] != "example.org" {
		t.Errorf("rows[0] = %v", rows[0])
	}
}

func TestMailingListSingularTabular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "mailinglist/get_mailinglists_response_success_single.xml")
	list, _ := mailinglist.DecodeMailingLists(resp.Body.ReturnInfo)
	if len(list) != 1 {
		t.Fatalf("len = %d, want 1", len(list))
	}
	// The order is part of the user-visible table contract: identity
	// (name, domain) before list config before lifecycle state. The
	// password is intentionally absent from table output. Pin it by
	// index so a refactor reordering the slice does not slip through.
	want := [][]string{
		{"mailinglist_name", "announce-example-com"},
		{"mailinglist_domain", "example.org"},
		{"mailinglist_is_active", "Y"},
		{"mailinglist_subscriber", ""},
		{"mailinglist_config", "# Mailinglist configuration"},
		{"mailinglist_restrict_post", ""},
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
