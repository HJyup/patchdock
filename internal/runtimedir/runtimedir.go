package runtimedir

import (
	"fmt"
	"os"
	"path/filepath"
)

type Dir struct {
	root string
}

func Resolve(root string) (Dir, error) {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return Dir{}, fmt.Errorf("create runtime dir %s: %w", root, err)
	}

	info, err := os.Stat(root)
	if err != nil {
		return Dir{}, fmt.Errorf("inspect runtime dir %s: %w", root, err)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		if err := os.Chmod(root, 0o700); err != nil {
			return Dir{}, fmt.Errorf("runtime dir %s has mode %o and cannot be restricted to 0700: %w", root, perm, err)
		}
	}

	return Dir{root: root}, nil
}

func (d Dir) Root() string {
	return d.root
}

func (d Dir) Socket() string {
	return filepath.Join(d.root, "dock.sock")
}

func (d Dir) Lock() string {
	return filepath.Join(d.root, "dock.lock")
}
