package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
)

type temporaryDir struct {
	rootDir string
}

func newTemporaryDir() (*temporaryDir, error) {
	tempIO, err := os.MkdirTemp("", "patchdock-io-*")
	if err != nil {
		return nil, err
	}
	return &temporaryDir{rootDir: tempIO}, nil
}

func (e *temporaryDir) Cleanup() {
	os.RemoveAll(e.rootDir)
}

func (e *temporaryDir) WorkspacePath() string {
	return filepath.Join(e.rootDir, "work")
}

func (e *temporaryDir) PlannerPath() string {
	return filepath.Join(e.rootDir, "planner")
}

func (e *temporaryDir) ExecutorPath(attempt int) string {
	return filepath.Join(e.rootDir, fmt.Sprintf("executor-%d", attempt))
}

func (e *temporaryDir) ReviewPath(attempt int) string {
	return filepath.Join(e.rootDir, fmt.Sprintf("review-%d", attempt))
}
