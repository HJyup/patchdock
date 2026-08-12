package lock

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func lockPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "dock.lock")
}

func mustAcquire(t *testing.T, path string) *Lock {
	t.Helper()

	l, err := Acquire(path)
	if err != nil {
		t.Fatalf("acquire %s: %v", path, err)
	}
	t.Cleanup(func() { _ = l.Release() })
	return l
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return strings.TrimSpace(string(raw))
}

func TestAcquireCreatesThePrivateLockFile(t *testing.T) {
	path := lockPath(t)

	mustAcquire(t, path)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("lock file was not created: %v", err)
	}

	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("permissions = %04o, want 0600", perm)
	}
}

func TestSecondAcquireIsRefusedWhileHeld(t *testing.T) {
	path := lockPath(t)
	mustAcquire(t, path)

	second, err := Acquire(path)
	if !errors.Is(err, ErrHeld) {
		t.Fatalf("second Acquire = %v, want %v", err, ErrHeld)
	}
	if second != nil {
		t.Error("Acquire returned a lock alongside an error")
	}
}

func TestReleaseAllowsReacquisition(t *testing.T) {
	path := lockPath(t)

	first, err := Acquire(path)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	if err := first.Release(); err != nil {
		t.Fatalf("release: %v", err)
	}

	second, err := Acquire(path)
	if err != nil {
		t.Fatalf("acquire after release: %v", err)
	}
	if err := second.Release(); err != nil {
		t.Fatalf("release second: %v", err)
	}
}

func TestAcquireRecordsTheOwningPID(t *testing.T) {
	path := lockPath(t)
	mustAcquire(t, path)

	if got := readFile(t, path); got != strconv.Itoa(os.Getpid()) {
		t.Errorf("lock file contains %q, want the pid %d", got, os.Getpid())
	}
}

func TestOwnerWhenNoLockFileExists(t *testing.T) {
	_, err := Owner(lockPath(t))
	if !errors.Is(err, ErrNotHeld) {
		t.Fatalf("Owner(missing) = %v, want %v", err, ErrNotHeld)
	}
}
