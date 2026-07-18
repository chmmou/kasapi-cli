package directoryprotection_test

import (
	"context"
	"errors"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/directoryprotection"
	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func TestDecodeDirectoryProtections(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "directoryprotection/get_directoryprotection_response_success_all.xml")
	got, err := directoryprotection.DecodeDirectoryProtections(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeDirectoryProtections: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	d := got[0]
	if d.User != "protected_user" {
		t.Errorf("User = %q", d.User)
	}
	if d.Path != "/protected/directory/" {
		t.Errorf("Path = %q", d.Path)
	}
	if d.AuthName != "ByPassword" {
		t.Errorf("AuthName = %q", d.AuthName)
	}
	if d.InProgress != "FALSE" {
		t.Errorf("InProgress = %q", d.InProgress)
	}
	// Second entry: a directory with multiple protected users surfaces
	// as multiple list rows (the read shape this module deliberately
	// exposes instead of a single Get).
	if got[1].User != "protected_user_1" || got[1].Path != "/protected/directory/1/" {
		t.Errorf("got[1] = %+v, want user=protected_user_1 path=/protected/directory/1/", got[1])
	}
}

func TestDecodeDirectoryProtectionSingular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "directoryprotection/get_directoryprotection_response_success.xml")
	got, err := directoryprotection.DecodeDirectoryProtections(resp.Body.ReturnInfo)
	if err != nil {
		t.Fatalf("DecodeDirectoryProtections: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Path != "/protected/directory/" {
		t.Errorf("Path = %q", got[0].Path)
	}
}

func TestClientListNoFilter(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "directoryprotection/get_directoryprotection_response_success_all.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	list, err := directoryprotection.NewClient(fc).List(context.Background(), "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if fc.GotAction != "get_directoryprotection" {
		t.Errorf("action = %q, want get_directoryprotection", fc.GotAction)
	}
	if fc.GotParams != nil {
		t.Errorf("params = %v, want nil", fc.GotParams)
	}
	if len(list) == 0 {
		t.Errorf("len = %d, want a non-empty list", len(list))
	}
}

func TestClientListWithPath(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "directoryprotection/get_directoryprotection_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	list, err := directoryprotection.NewClient(fc).List(context.Background(), "/protected/directory/")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if fc.GotAction != "get_directoryprotection" {
		t.Errorf("action = %q", fc.GotAction)
	}
	if got, _ := fc.GotParams["directory_path"].(string); got != "/protected/directory/" {
		t.Errorf("params[directory_path] = %v", fc.GotParams["directory_path"])
	}
	if len(list) == 0 {
		t.Errorf("len = %d, want a non-empty list", len(list))
	}
}

func TestClientPropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := directoryprotection.NewClient(&testutil.FakeCaller{Err: want})
	if _, err := c.List(context.Background(), ""); !errors.Is(err, want) {
		t.Errorf("List err = %v, want %v wrapped", err, want)
	}
	if _, err := c.List(context.Background(), "/foo/"); !errors.Is(err, want) {
		t.Errorf("List(path) err = %v, want %v wrapped", err, want)
	}
}

func TestDirectoryProtectionListTabular(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "directoryprotection/get_directoryprotection_response_success_all.xml")
	list, _ := directoryprotection.DecodeDirectoryProtections(resp.Body.ReturnInfo)
	headers := list.TableHeaders()
	if headers[0] != "PATH" {
		t.Errorf("headers[0] = %q, want PATH", headers[0])
	}
	rows := list.TableRows()
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	if rows[0][0] != "/protected/directory/" {
		t.Errorf("rows[0][PATH] = %q", rows[0][0])
	}
}
