package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/cli"
)

func TestUsageTrafficRejectsInvalidFlags(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"month negative", []string{"usage", "traffic", "--month", "-1"}, "--month must be between 1 and 12"},
		{"month too high", []string{"usage", "traffic", "--month", "13"}, "--month must be between 1 and 12"},
		{"year too low", []string{"usage", "traffic", "--year", "1999"}, "--year must be between"},
		{"year typo", []string{"usage", "traffic", "--year", "20256"}, "--year must be between"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root, opts := cli.NewRootCmd()
			root.AddCommand(cli.NewUsageCmd(opts))

			var buf bytes.Buffer
			root.SetOut(&buf)
			root.SetErr(&buf)
			root.SetArgs(tc.args)
			err := root.Execute()
			if err == nil {
				t.Fatalf("Execute returned nil, want error containing %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want substring %q", err, tc.want)
			}
		})
	}
}
