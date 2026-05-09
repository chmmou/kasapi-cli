// Package testutil provides shared helpers used by tests across the
// internal/ packages: repository-root discovery, SOAP fixture loading,
// and a fake Caller stub for module Client tests.
//
// All helpers depend only on the testing package and on internal/soap,
// so any external test package (package <pkg>_test) under internal/ can
// import this package without creating a cycle.
package testutil

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/chmmou/kasapi-cli/internal/soap"
)

// RepoRoot returns the absolute path to the repository root by climbing
// from this file's location until a go.mod is found. Tests use it to
// resolve fixtures under the repo-level testdata/ directory.
func RepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("repo root not found from %q", file)
		}
		dir = parent
	}
}

// DecodeFixture opens testdata/<relPath> and decodes the SOAP envelope.
// relPath is forward-slash separated relative to the repo's testdata/
// directory (e.g. "mailinglist/get_mailinglists_response_success.xml").
func DecodeFixture(t *testing.T, relPath string) *soap.Response {
	t.Helper()
	path := filepath.Join(RepoRoot(t), "testdata", filepath.FromSlash(relPath))
	//nolint:gosec // G304: test fixture loader, path is rooted at RepoRoot(t).
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", relPath, err)
	}
	defer func() { _ = f.Close() }()
	resp, err := soap.Decode(f)
	if err != nil {
		t.Fatalf("decode %s: %v", relPath, err)
	}
	return resp
}

// FakeCaller is a minimal stub for the Caller interface implemented by
// every internal/<module>.Client. It returns the configured Resp/Err
// verbatim and records the most recent action and params for assertions.
type FakeCaller struct {
	Resp *soap.Response
	Err  error

	GotAction string
	GotParams map[string]any
}

// Call satisfies the per-module Caller interface (Call(ctx, action,
// params) (*soap.Response, error)).
func (f *FakeCaller) Call(_ context.Context, action string, params map[string]any) (*soap.Response, error) {
	f.GotAction = action
	f.GotParams = params
	return f.Resp, f.Err
}
