package session_test

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/chmmou/kasapi-cli/internal/session"
)

func newStore(t *testing.T, now time.Time) *session.Store {
	t.Helper()
	s, err := session.New(filepath.Join(t.TempDir(), "sessions.toml"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	s.Now = func() time.Time { return now }
	return s
}

func TestSaveLoadRoundTrip(t *testing.T) {
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	s := newStore(t, now)
	want := session.Entry{
		Token:           "tok-abc",
		ExpiresAt:       now.Add(time.Hour),
		LifetimeSeconds: 3600,
		UpdateLifetime:  true,
	}
	if err := s.Save("w0000000", want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load("w0000000")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got == nil {
		t.Fatal("Load: nil entry")
	}
	if got.Token != want.Token || !got.ExpiresAt.Equal(want.ExpiresAt) ||
		got.LifetimeSeconds != want.LifetimeSeconds || got.UpdateLifetime != want.UpdateLifetime {
		t.Errorf("Load = %+v, want %+v", got, want)
	}
}

func TestSaveComputesExpiresAtFromLifetime(t *testing.T) {
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	s := newStore(t, now)
	if err := s.Save("w0", session.Entry{Token: "t", LifetimeSeconds: 600}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load("w0")
	if err != nil || got == nil {
		t.Fatalf("Load: %v %v", got, err)
	}
	want := now.Add(10 * time.Minute)
	if !got.ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt = %v, want %v", got.ExpiresAt, want)
	}
}

func TestSaveFallsBackToDefaultLifetime(t *testing.T) {
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	s := newStore(t, now)
	if err := s.Save("w0", session.Entry{Token: "t"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load("w0")
	if err != nil || got == nil {
		t.Fatalf("Load: %v %v", got, err)
	}
	want := now.Add(session.DefaultLifetime)
	if !got.ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt = %v, want %v", got.ExpiresAt, want)
	}
}

func TestLoadReturnsNilWhenFileMissing(t *testing.T) {
	s := newStore(t, time.Now())
	got, err := s.Load("anything")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != nil {
		t.Errorf("Load = %v, want nil", got)
	}
}

func TestLoadReturnsNilWhenLoginAbsent(t *testing.T) {
	s := newStore(t, time.Now())
	if err := s.Save("w0", session.Entry{Token: "t", LifetimeSeconds: 60}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load("other")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != nil {
		t.Errorf("Load other = %v, want nil", got)
	}
}

func TestLoadDropsExpiredEntry(t *testing.T) {
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	s := newStore(t, now)
	if err := s.Save("w0", session.Entry{
		Token:     "t",
		ExpiresAt: now.Add(-time.Second),
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load("w0")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != nil {
		t.Errorf("expected expired Load to return nil, got %+v", got)
	}
	// File should be gone since it had only one entry.
	if _, err := s.Load("w0"); err != nil {
		t.Errorf("second Load on cleaned file: %v", err)
	}
}

func TestDeleteRemovesEntryOnly(t *testing.T) {
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	s := newStore(t, now)
	if err := s.Save("a", session.Entry{Token: "1", LifetimeSeconds: 60}); err != nil {
		t.Fatalf("Save a: %v", err)
	}
	if err := s.Save("b", session.Entry{Token: "2", LifetimeSeconds: 60}); err != nil {
		t.Fatalf("Save b: %v", err)
	}
	if err := s.Delete("a"); err != nil {
		t.Fatalf("Delete a: %v", err)
	}
	if got, _ := s.Load("a"); got != nil {
		t.Errorf("a still present after Delete")
	}
	if got, _ := s.Load("b"); got == nil {
		t.Errorf("b should still be present")
	}
}

func TestDeleteMissingIsNoop(t *testing.T) {
	s := newStore(t, time.Now())
	if err := s.Delete("nope"); err != nil {
		t.Errorf("Delete on missing file: %v", err)
	}
}

func TestPathForCustomConfig(t *testing.T) {
	got, err := session.PathFor("/tmp/custom/config.toml")
	if err != nil {
		t.Fatalf("PathFor: %v", err)
	}
	want := "/tmp/custom/sessions.toml"
	if got != want {
		t.Errorf("PathFor = %q, want %q", got, want)
	}
}

func TestPathForEmptyFallsBackToDefault(t *testing.T) {
	got, err := session.PathFor("")
	if err != nil {
		t.Fatalf("PathFor: %v", err)
	}
	def, err := session.DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	if got != def {
		t.Errorf("PathFor(\"\") = %q, want %q", got, def)
	}
}

func TestSaveWritesMode0600(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file mode bits not meaningful on Windows")
	}
	s := newStore(t, time.Now())
	if err := s.Save("w0", session.Entry{Token: "t", LifetimeSeconds: 60}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	info, err := statFile(s.Path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("mode = %#o, want 0600", mode)
	}
}
