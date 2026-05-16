package server_test

import (
	"context"
	"errors"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/server"
	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestDecodeServices(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "account/get_server_information_response_success.xml")
	got, err := server.DecodeServices(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeServices: %v", err)
	}
	if len(got) != 8 {
		t.Fatalf("len = %d, want 8", len(got))
	}
	if got[0].Service != "mysql" || got[0].Version != "10.6.12" || got[0].VersionType != "server" {
		t.Errorf("got[0] = %+v", got[0])
	}
	if got[1].Service != "php" || got[1].FileExtension != "php56" || got[1].Interface != "cgi-fcgi" {
		t.Errorf("got[1] = %+v", got[1])
	}
	last := got[len(got)-1]
	if last.Service != "os" || last.Distribution != "ubuntu" || last.Version != "22" {
		t.Errorf("os entry = %+v", last)
	}
}

func TestClientInformation(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "account/get_server_information_response_success.xml")
	c := server.NewClient(&testutil.FakeCaller{Resp: resp})
	list, err := c.Information(context.Background())
	if err != nil {
		t.Fatalf("Information: %v", err)
	}
	if len(list) == 0 {
		t.Errorf("len = %d, want a non-empty list", len(list))
	}
}

func TestClientInformationPropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := server.NewClient(&testutil.FakeCaller{Err: want})
	if _, err := c.Information(context.Background()); !errors.Is(err, want) {
		t.Errorf("err = %v, want %v wrapped", err, want)
	}
}

func TestServiceListTabular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "account/get_server_information_response_success.xml")
	list, _ := server.DecodeServices(resp.Body.ReturnInfo)
	rows := list.TableRows()
	if len(rows) != 8 {
		t.Errorf("rows = %d, want 8", len(rows))
	}
	if rows[0][0] != "mysql" || rows[0][4] != "server" {
		t.Errorf("rows[0] = %v", rows[0])
	}
}
