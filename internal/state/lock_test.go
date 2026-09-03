package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAcquireAndRelease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	lock, err := Acquire(path)
	if err != nil {
		t.Fatalf("unexpected error acquiring lock: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected lock file to exist: %v", err)
	}

	if err := lock.Release(); err != nil {
		t.Fatalf("unexpected error releasing lock: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected lock file to be removed after release")
	}
}

func TestConcurrentAcquireIsRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	first, err := Acquire(path)
	if err != nil {
		t.Fatalf("unexpected error on first acquire: %v", err)
	}
	defer first.Release()

	_, err = Acquire(path)
	if err != ErrLocked {
		t.Fatalf("expected ErrLocked on concurrent acquire, got %v", err)
	}
}

func TestInspectMissingLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	info, found, err := Inspect(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found || info != nil {
		t.Fatalf("expected no lock found, got %+v", info)
	}
}

func TestInspectAndStaleDetectionWithDeadPID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	// A PID astronomically unlikely to be alive on any real system, and
	// definitely not this test process.
	if err := os.WriteFile(path, []byte("999999999\n2020-01-01T00:00:00Z\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	info, found, err := Inspect(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Fatal("expected lock to be found")
	}
	if info.PID != 999999999 {
		t.Errorf("expected PID 999999999, got %d", info.PID)
	}

	if !IsStale(info) {
		t.Errorf("expected lock with a non-existent PID to be reported stale")
	}
}

func TestInspectLiveProcessIsNotStale(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	lock, err := Acquire(path)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()

	info, found, err := Inspect(path)
	if err != nil || !found {
		t.Fatalf("expected to find lock, err=%v found=%v", err, found)
	}
	if IsStale(info) {
		t.Errorf("expected the current test process's own lock to not be stale")
	}
}

func TestForceReleaseRemovesLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	if err := os.WriteFile(path, []byte("999999999\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ForceRelease(path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected lock file removed")
	}
}

func TestForceReleaseMissingLockIsNoop(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	if err := ForceRelease(path); err != nil {
		t.Fatalf("expected no error force-releasing a missing lock, got %v", err)
	}
}
