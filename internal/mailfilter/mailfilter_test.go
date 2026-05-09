package mailfilter_test

import (
	"context"
	"errors"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/mailfilter"
	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestDecodeStandardFilters(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "mailfilter/get_mailstandardfilter_response_success.xml")
	got, err := mailfilter.DecodeStandardFilters(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeStandardFilters: %v", err)
	}
	if len(got) != 9 {
		t.Fatalf("len = %d, want 9 (per fixture arrayType)", len(got))
	}
	if got[0].Filter != "rspamd" || got[0].Type != "rspamd" || got[0].Recommended != "Y" {
		t.Errorf("got[0] = %+v", got[0])
	}
	// Spot-check an entry whose type differs from the filter id.
	for _, f := range got {
		if f.Filter == "pdw" {
			if f.Type != "reject" || f.Title != "policyd-weight" {
				t.Errorf("pdw entry = %+v", f)
			}
			return
		}
	}
	t.Errorf("pdw entry missing")
}

func TestClientList(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "mailfilter/get_mailstandardfilter_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	list, err := mailfilter.NewClient(fc).List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if fc.GotAction != "get_mailstandardfilter" {
		t.Errorf("action = %q, want get_mailstandardfilter", fc.GotAction)
	}
	if fc.GotParams != nil {
		t.Errorf("params = %v, want nil", fc.GotParams)
	}
	if len(list) != 9 {
		t.Errorf("len = %d, want 9", len(list))
	}
}

func TestClientPropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := mailfilter.NewClient(&testutil.FakeCaller{Err: want})
	if _, err := c.List(context.Background()); !errors.Is(err, want) {
		t.Errorf("List err = %v, want %v wrapped", err, want)
	}
}

func TestStandardFilterListTabular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "mailfilter/get_mailstandardfilter_response_success.xml")
	list, _ := mailfilter.DecodeStandardFilters(resp.Body.ReturnInfo)
	rows := list.TableRows()
	if len(rows) != 9 {
		t.Fatalf("rows = %d, want 9", len(rows))
	}
	if rows[0][0] != "rspamd" {
		t.Errorf("rows[0][0] = %q, want rspamd", rows[0][0])
	}
}
