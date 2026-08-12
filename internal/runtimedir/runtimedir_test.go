package runtimedir

import (
	"os"
	"path/filepath"
	"testing"
)

func mustResolve(t *testing.T, root string) Dir {
	t.Helper()

	dir, err := Resolve(root)
	if err != nil {
		t.Fatalf("resolve %s: %v", root, err)
	}
	return dir
}

func assertPerm(t *testing.T, path string, want os.FileMode) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Errorf("%s has mode %04o, want %04o", path, got, want)
	}
}

func TestResolveCreatesThePrivateDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "patchdock")

	dir := mustResolve(t, root)

	if dir.Root() != root {
		t.Errorf("Root() = %q, want %q", dir.Root(), root)
	}
	assertPerm(t, root, 0o700)
}

func TestResolveIsIdempotent(t *testing.T) {
	root := filepath.Join(t.TempDir(), "patchdock")

	first := mustResolve(t, root)
	second := mustResolve(t, root)

	if first != second {
		t.Errorf("second resolve = %+v, want %+v", second, first)
	}
	assertPerm(t, root, 0o700)
}

func TestPathsLiveUnderRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "patchdock")
	dir := mustResolve(t, root)

	paths := map[string]string{
		"Socket": dir.Socket(),
		"Config": dir.Config(),
		"Lock":   dir.Lock(),
		"Log":    dir.Log(),
	}

	seen := make(map[string]string, len(paths))
	for name, path := range paths {
		if got := filepath.Dir(path); got != root {
			t.Errorf("%s() = %q, want it inside %q", name, path, root)
		}
		if other, clash := seen[path]; clash {
			t.Errorf("%s() and %s() both resolve to %q", name, other, path)
		}
		seen[path] = name
	}
}

func TestPathNames(t *testing.T) {
	dir := mustResolve(t, filepath.Join(t.TempDir(), "patchdock"))

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"Socket", dir.Socket(), "dock.sock"},
		{"Config", dir.Config(), "config.json"},
		{"Lock", dir.Lock(), "dock.lock"},
		{"Log", dir.Log(), "dock.log"},
	}

	for _, tt := range tests {
		if base := filepath.Base(tt.got); base != tt.want {
			t.Errorf("%s() base = %q, want %q", tt.name, base, tt.want)
		}
	}
}
