package cli_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/cli"
)

func TestExitErrorWraps(t *testing.T) {
	t.Parallel()
	base := errors.New("disk full")
	ee := cli.UserError(base, "open config")
	if !errors.Is(ee, base) {
		t.Error("UserError should wrap the inner error so errors.Is works")
	}
	if got, want := ee.Error(), "open config: disk full"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
	if ee.Code != cli.ExitUserError {
		t.Errorf("Code = %d, want %d", ee.Code, cli.ExitUserError)
	}
}

func TestUserErrorWithoutPrefix(t *testing.T) {
	t.Parallel()
	ee := cli.UserError(errors.New("bad"), "")
	if got, want := ee.Error(), "bad"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestAPIErrorCode(t *testing.T) {
	t.Parallel()
	ee := cli.APIError(errors.New("kas-api fault"), "call accounts.get")
	if ee.Code != cli.ExitAPIError {
		t.Errorf("Code = %d, want %d", ee.Code, cli.ExitAPIError)
	}
}

func TestCodeFor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		err  error
		want int
	}{
		{nil, cli.ExitOK},
		{cli.UserError(errors.New("x"), ""), cli.ExitUserError},
		{cli.APIError(errors.New("x"), ""), cli.ExitAPIError},
		{fmt.Errorf("wrapped: %w", cli.UserError(errors.New("x"), "")), cli.ExitUserError},
		{errors.New("plain"), cli.ExitAPIError},
	}
	for _, tt := range tests {
		if got := cli.CodeFor(tt.err); got != tt.want {
			t.Errorf("CodeFor(%v) = %d, want %d", tt.err, got, tt.want)
		}
	}
}
