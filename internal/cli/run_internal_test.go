package cli

// White-box tests for the runWriteE seam: the success-audit created_id
// correlation and the refusal→outcome mapping are internal wiring the
// cli_test package cannot reach (writeSpec and refusalOutcome are
// unexported), so they are pinned here.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/api"
)

// TestRunWriteESuccessAuditCarriesCreatedID pins that a create action
// whose identifier KAS generates server-side surfaces that identifier
// in the success audit record (created_id), not only in the rendered
// success line.
func TestRunWriteESuccessAuditCarriesCreatedID(t *testing.T) {
	// No t.Parallel(): t.Setenv forbids it. The env override and the
	// nonexistent --config path keep the test hermetic — without them
	// resolveCreds would read the developer's real config file and a
	// host-set KAS_AUDIT_LOG would make runWriteE append to the real
	// audit log.
	t.Setenv("KAS_AUDIT_LOG", "")
	opts := &RootOptions{
		ConfigPath: filepath.Join(t.TempDir(), "config.toml"),
		Login:      "w0000000",
		AuthData:   "x",
		AuthType:   "plain",
		Output:     FormatTable,
	}
	var createdID string
	run := runWriteE(opts, func([]string) (writeSpec, error) {
		return writeSpec{
			action:      "add_ftpuser",
			destructive: false,
			confirm:     ConfirmAction{Verb: "create", Resource: "ftp user", ID: "backup user"},
			params:      map[string]any{"ftp_comment": "backup user"},
			createdID:   &createdID,
			dispatch: func(_ *api.Client, _ context.Context) (string, error) {
				createdID = "f0000001"
				return "created ftp user f0000001", nil
			},
		}, nil
	})
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	if err := run(cmd, nil); err != nil {
		t.Fatalf("runWriteE: %v", err)
	}
	if !strings.Contains(errb.String(), "created_id=f0000001") {
		t.Errorf("audit line missing created_id=f0000001: %s", errb.String())
	}
	if !strings.Contains(errb.String(), "outcome=success") {
		t.Errorf("audit line missing outcome=success: %s", errb.String())
	}
}

func TestRefusalOutcome(t *testing.T) {
	t.Parallel()
	cases := []struct {
		err  error
		want string
	}{
		{UserError(ErrConfirmationDeclined, ""), AuditOutcomeDeclined},
		{UserError(ErrConfirmationRequired, ""), AuditOutcomeRefused},
		{UserError(fmt.Errorf("%w: %w", ErrConfirmationAborted, errors.New("tty gone")), ""), AuditOutcomeAborted},
		{errors.New("unrelated"), ""},
		{nil, ""},
	}
	for _, c := range cases {
		if got := refusalOutcome(c.err); got != c.want {
			t.Errorf("refusalOutcome(%v) = %q, want %q", c.err, got, c.want)
		}
	}
}
