package session_test

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gofrs/flock"

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
	if err := s.Save(t.Context(), "w0000000", want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load(t.Context(), "w0000000")
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
	if err := s.Save(t.Context(), "w0", session.Entry{Token: "t", LifetimeSeconds: 600}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load(t.Context(), "w0")
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
	if err := s.Save(t.Context(), "w0", session.Entry{Token: "t"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load(t.Context(), "w0")
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
	got, err := s.Load(t.Context(), "anything")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != nil {
		t.Errorf("Load = %v, want nil", got)
	}
}

func TestLoadReturnsNilWhenLoginAbsent(t *testing.T) {
	s := newStore(t, time.Now())
	if err := s.Save(t.Context(), "w0", session.Entry{Token: "t", LifetimeSeconds: 60}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load(t.Context(), "other")
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
	if err := s.Save(t.Context(), "w0", session.Entry{
		Token:     "t",
		ExpiresAt: now.Add(-time.Second),
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load(t.Context(), "w0")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != nil {
		t.Errorf("expected expired Load to return nil, got %+v", got)
	}
	// File should be gone since it had only one entry.
	if _, err := s.Load(t.Context(), "w0"); err != nil {
		t.Errorf("second Load on cleaned file: %v", err)
	}
}

func TestDeleteRemovesEntryOnly(t *testing.T) {
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	s := newStore(t, now)
	if err := s.Save(t.Context(), "a", session.Entry{Token: "1", LifetimeSeconds: 60}); err != nil {
		t.Fatalf("Save a: %v", err)
	}
	if err := s.Save(t.Context(), "b", session.Entry{Token: "2", LifetimeSeconds: 60}); err != nil {
		t.Fatalf("Save b: %v", err)
	}
	if err := s.Delete(t.Context(), "a"); err != nil {
		t.Fatalf("Delete a: %v", err)
	}
	if got, _ := s.Load(t.Context(), "a"); got != nil {
		t.Errorf("a still present after Delete")
	}
	if got, _ := s.Load(t.Context(), "b"); got == nil {
		t.Errorf("b should still be present")
	}
}

func TestDeleteMissingIsNoop(t *testing.T) {
	s := newStore(t, time.Now())
	if err := s.Delete(t.Context(), "nope"); err != nil {
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
	if err := s.Save(t.Context(), "w0", session.Entry{Token: "t", LifetimeSeconds: 60}); err != nil {
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

// TestSaveBlocksWhileLockHeld pins the cross-process serialisation
// guarantee from the advisory lock at LockPath: while another holder
// has the lock, Save must block instead of racing the read-modify-
// write. Same-process fd-distinct flocks serve as a proxy for
// cross-process behaviour — flock(2) is per-fd on Linux/macOS, so
// holding the lock from a different *flock.Flock instance produces
// the same blocking that two separate processes would.
func TestSaveBlocksWhileLockHeld(t *testing.T) {
	s := newStore(t, time.Now())

	// Force the lock-file directory + path to exist by running one Save
	// first; otherwise the external flock.New below would lock a path
	// that the production code creates on demand.
	if err := s.Save(t.Context(), "seed", session.Entry{Token: "t", LifetimeSeconds: 60}); err != nil {
		t.Fatalf("seed Save: %v", err)
	}

	ext := flock.New(s.LockPath())
	if err := ext.Lock(); err != nil {
		t.Fatalf("external Lock: %v", err)
	}
	released := false
	t.Cleanup(func() {
		if !released {
			_ = ext.Unlock()
		}
	})

	done := make(chan error, 1)
	go func() {
		done <- s.Save(t.Context(), "w0", session.Entry{Token: "blocked", LifetimeSeconds: 60})
	}()

	select {
	case err := <-done:
		t.Fatalf("Save returned %v while external lock was held; expected to block", err)
	case <-time.After(100 * time.Millisecond):
		// expected: Save is blocked on the lock
	}

	if err := ext.Unlock(); err != nil {
		t.Fatalf("external Unlock: %v", err)
	}
	released = true

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Save after Unlock: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Save did not complete after external Unlock")
	}

	got, err := s.Load(t.Context(), "w0")
	if err != nil || got == nil || got.Token != "blocked" {
		t.Fatalf("post-unlock Load = %+v, %v; want token=blocked", got, err)
	}
}

// TestStoreReleasesLockAfterSave verifies the lock is released once
// Save returns, so the next call from any process can take it again
// without timing out.
func TestStoreReleasesLockAfterSave(t *testing.T) {
	s := newStore(t, time.Now())
	if err := s.Save(t.Context(), "w0", session.Entry{Token: "t", LifetimeSeconds: 60}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	ext := flock.New(s.LockPath())
	locked, err := ext.TryLock()
	if err != nil {
		t.Fatalf("TryLock: %v", err)
	}
	if !locked {
		t.Fatal("lock not released after Save returned")
	}
	_ = ext.Unlock()
}
