package cronjob_test

import (
	"context"
	"errors"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/cronjob"
	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/testutil"
)

func sampleSpec() cronjob.Spec {
	return cronjob.Spec{
		Protocol:      "https",
		HTTPURL:       "example.de/cron.php",
		Comment:       "Hourly Cron",
		Minute:        "59",
		Hour:          "*",
		DayOfMonth:    "*",
		Month:         "*",
		DayOfWeek:     "*",
		MailAdress:    "cronjob@example.de",
		MailCondition: "no-mail",
		MailSubject:   "comment",
		IsActive:      "Y",
	}
}

func TestClientAdd(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "cronjob/add_cronjob_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	id, err := cronjob.NewClient(fc).Add(context.Background(), sampleSpec())
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if fc.GotAction != "add_cronjob" {
		t.Errorf("action = %q, want add_cronjob", fc.GotAction)
	}
	if fc.GotParams["protocol"] != "https" || fc.GotParams["http_url"] != "example.de/cron.php" ||
		fc.GotParams["cronjob_comment"] != "Hourly Cron" || fc.GotParams["minute"] != "59" {
		t.Errorf("params = %v", fc.GotParams)
	}
	if fc.GotParams["mail_adress"] != "cronjob@example.de" {
		t.Errorf("mail_adress param = %v, want cronjob@example.de (single-d wire key)", fc.GotParams["mail_adress"])
	}
	if id != "324700" {
		t.Errorf("returned id = %q, want 324700 (fixture ReturnInfo)", id)
	}
}

// A success-with-warning response (e.g. the unconfirmed-email notice)
// still carries ReturnString=TRUE, so Add must treat it as success and
// return the new id rather than wrapping ErrUnexpectedReturnString.
func TestClientAddWarningStillSucceeds(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "cronjob/add_cronjob_response_warning.xml")
	id, err := cronjob.NewClient(&testutil.FakeCaller{Resp: resp}).Add(context.Background(), sampleSpec())
	if err != nil {
		t.Fatalf("Add (warning): %v", err)
	}
	if id != "324700" {
		t.Errorf("returned id = %q, want 324700 (warning fixture ReturnInfo)", id)
	}
}

func TestClientUpdate(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "cronjob/update_cronjob_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	fields := map[string]string{
		cronjob.FieldComment: "Every hour Cron",
		cronjob.FieldMinute:  "1",
	}
	if err := cronjob.NewClient(fc).Update(context.Background(), "324700", fields); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if fc.GotAction != "update_cronjob" {
		t.Errorf("action = %q, want update_cronjob", fc.GotAction)
	}
	if fc.GotParams["cronjob_id"] != "324700" ||
		fc.GotParams["cronjob_comment"] != "Every hour Cron" || fc.GotParams["minute"] != "1" {
		t.Errorf("params = %v", fc.GotParams)
	}
}

func TestClientDelete(t *testing.T) {
	t.Parallel()
	resp := testutil.DecodeFixture(t, "cronjob/delete_cronjob_response_success.xml")
	fc := &testutil.FakeCaller{Resp: resp}
	if err := cronjob.NewClient(fc).Delete(context.Background(), "324700"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if fc.GotAction != "delete_cronjob" {
		t.Errorf("action = %q, want delete_cronjob", fc.GotAction)
	}
	if fc.GotParams["cronjob_id"] != "324700" {
		t.Errorf("params = %v", fc.GotParams)
	}
}

func TestWriteValidation(t *testing.T) {
	t.Parallel()
	c := cronjob.NewClient(&testutil.FakeCaller{})
	ctx := context.Background()

	missingURL := sampleSpec()
	missingURL.HTTPURL = ""
	if _, err := c.Add(ctx, missingURL); err == nil {
		t.Error("Add missing http_url: err = nil, want validation error")
	}
	missingMinute := sampleSpec()
	missingMinute.Minute = ""
	if _, err := c.Add(ctx, missingMinute); err == nil {
		t.Error("Add missing minute: err = nil, want validation error")
	}
	missingHour := sampleSpec()
	missingHour.Hour = ""
	if _, err := c.Add(ctx, missingHour); err == nil {
		t.Error("Add missing hour: err = nil, want validation error")
	}
	if err := c.Update(ctx, "", map[string]string{cronjob.FieldComment: "x"}); err == nil {
		t.Error("Update empty id: err = nil, want validation error")
	}
	if err := c.Update(ctx, "324700", nil); err == nil {
		t.Error("Update no fields: err = nil, want validation error")
	}
	if err := c.Delete(ctx, ""); err == nil {
		t.Error("Delete empty id: err = nil, want validation error")
	}
}

func TestUnexpectedReturnString(t *testing.T) {
	t.Parallel()
	resp := &soap.Response{Body: soap.ResponseBody{ReturnString: "FALSE"}}
	c := cronjob.NewClient(&testutil.FakeCaller{Resp: resp})
	ctx := context.Background()
	if _, err := c.Add(ctx, sampleSpec()); !errors.Is(err, cronjob.ErrUnexpectedReturnString) {
		t.Errorf("Add err = %v, want ErrUnexpectedReturnString", err)
	}
	if err := c.Update(ctx, "324700", map[string]string{cronjob.FieldComment: "x"}); !errors.Is(err, cronjob.ErrUnexpectedReturnString) {
		t.Errorf("Update err = %v, want ErrUnexpectedReturnString", err)
	}
	if err := c.Delete(ctx, "324700"); !errors.Is(err, cronjob.ErrUnexpectedReturnString) {
		t.Errorf("Delete err = %v, want ErrUnexpectedReturnString", err)
	}
}

func TestWritePropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	c := cronjob.NewClient(&testutil.FakeCaller{Err: want})
	ctx := context.Background()
	if _, err := c.Add(ctx, sampleSpec()); !errors.Is(err, want) {
		t.Errorf("Add err = %v, want %v", err, want)
	}
	if err := c.Update(ctx, "324700", map[string]string{cronjob.FieldComment: "x"}); !errors.Is(err, want) {
		t.Errorf("Update err = %v, want %v", err, want)
	}
	if err := c.Delete(ctx, "324700"); !errors.Is(err, want) {
		t.Errorf("Delete err = %v, want %v", err, want)
	}
}

func TestParamBuilders(t *testing.T) {
	t.Parallel()
	add := cronjob.AddParams(sampleSpec())
	if add["protocol"] != "https" || add["http_url"] != "example.de/cron.php" ||
		add["cronjob_comment"] != "Hourly Cron" || add["minute"] != "59" ||
		add["mail_adress"] != "cronjob@example.de" || add["is_active"] != "Y" {
		t.Errorf("AddParams = %v", add)
	}
	if len(add) != 14 {
		t.Errorf("AddParams has %d keys, want 14 (all add_cronjob request fields)", len(add))
	}
	upd := cronjob.UpdateParams("324700", map[string]string{cronjob.FieldComment: "c", cronjob.FieldHour: "2"})
	if upd["cronjob_id"] != "324700" || upd["cronjob_comment"] != "c" || upd["hour"] != "2" {
		t.Errorf("UpdateParams = %v", upd)
	}
	del := cronjob.DeleteParams("324700")
	if len(del) != 1 || del["cronjob_id"] != "324700" {
		t.Errorf("DeleteParams = %v", del)
	}
}
