package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ErrLocked is returned by Acquire when another Howl operation currently
// holds the lock.
var ErrLocked = errors.New("another Howl installation operation is currently active")

// Lock represents a held installation lock. Call Release when the
// operation is done.
type Lock struct {
	path string
}

// Acquire takes the installation lock at path, failing with ErrLocked if
// another process already holds it. It never removes an existing lock
// file itself, even a stale one -- that's a `howl doctor --fix` decision,
// not something a mutating command does silently.
func Acquire(path string) (*Lock, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create lock directory: %w", err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return nil, ErrLocked
		}
		return nil, fmt.Errorf("failed to acquire installation lock: %w", err)
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "%d\n%s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		os.Remove(path)
		return nil, fmt.Errorf("failed to write lock metadata: %w", err)
	}

	return &Lock{path: path}, nil
}

// Release drops the lock. Safe to call once; the lock file is removed.
func (l *Lock) Release() error {
	if l == nil {
		return nil
	}
	if err := os.Remove(l.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to release installation lock: %w", err)
	}
	return nil
}

// LockInfo describes the contents of an existing lock file.
type LockInfo struct {
	PID        int
	AcquiredAt string
}

// Inspect reads an existing lock file's metadata without acquiring it.
// Returns (nil, false, nil) if no lock file exists.
func Inspect(path string) (*LockInfo, bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("failed to read lock file: %w", err)
	}

	lines := strings.SplitN(strings.TrimSpace(string(data)), "\n", 2)
	pid, convErr := strconv.Atoi(strings.TrimSpace(lines[0]))
	if convErr != nil {
		return nil, true, fmt.Errorf("lock file at %s is malformed", path)
	}
	info := &LockInfo{PID: pid}
	if len(lines) > 1 {
		info.AcquiredAt = strings.TrimSpace(lines[1])
	}
	return info, true, nil
}

// IsStale reports whether the process that acquired the lock is no longer
// running. Only meaningful on platforms where process liveness can be
// checked; on other platforms it conservatively reports false (not stale)
// so `doctor --fix` never removes a lock it can't actually verify.
func IsStale(info *LockInfo) bool {
	return !processAlive(info.PID)
}

// ForceRelease removes a lock file after the caller (doctor --fix) has
// confirmed it's safe to do so.
func ForceRelease(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove lock file: %w", err)
	}
	return nil
}
