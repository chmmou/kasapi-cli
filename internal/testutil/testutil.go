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
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

// AssertFaultFixtures binds a module's captured fault fixtures to the
// KAS contract: every testdata/<module>/*_response_failed_*.xml must
// decode to a *soap.FaultError with a non-empty Fault.String, and each
// entry in want (fixture filename -> expected fault code) must match
// exactly. It is the fixture<->contract anchor every module reuses so a
// captured fault fixture cannot silently drift from its documented KAS
// code. module is the testdata/ subdirectory (e.g. "ftpuser"); the
// empty string scans the shared top-level response_failed_*.xml set.
// want may be nil to assert only the universal invariant; pin a few
// representative documented codes there to also catch a code drift.
func AssertFaultFixtures(t *testing.T, module string, want map[string]string) {
	t.Helper()
	dir := filepath.Join(RepoRoot(t), "testdata", module)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read fixture dir %s: %v", dir, err)
	}
	seen := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		moduleFault := module != "" && strings.Contains(name, "_response_failed_")
		sharedFault := module == "" && strings.HasPrefix(name, "response_failed_") &&
			strings.HasSuffix(name, ".xml")
		if !moduleFault && !sharedFault {
			continue
		}
		seen++
		//nolint:gosec // G304: fixture path is rooted at the repo testdata/ dir.
		f, oerr := os.Open(filepath.Join(dir, name))
		if oerr != nil {
			t.Fatalf("open %s: %v", name, oerr)
		}
		_, derr := soap.Decode(f)
		_ = f.Close()
		var fe *soap.FaultError
		if !errors.As(derr, &fe) {
			t.Errorf("%s: decode err = %v, want *soap.FaultError", name, derr)
			continue
		}
		if fe.Fault.String == "" {
			t.Errorf("%s: empty fault code", name)
		}
		if code, ok := want[name]; ok && fe.Fault.String != code {
			t.Errorf("%s: fault = %q, want %q", name, fe.Fault.String, code)
		}
	}
	if seen == 0 {
		t.Fatalf("no fault fixtures found for module %q", module)
	}
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
