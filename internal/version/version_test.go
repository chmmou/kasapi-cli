package version

import (
	"strings"
	"testing"
)

func TestStringContainsFields(t *testing.T) {
	got := String()
	for _, want := range []string{"kasapi-cli", Version, Commit, Date} {
		if !strings.Contains(got, want) {
			t.Errorf("String() = %q, want it to contain %q", got, want)
		}
	}
}
