package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
)

type taskDir struct {
	rootDir string
}

func newDir() (*taskDir, error) {
	tempIO, err := os.MkdirTemp("", "patchdock-io-*")
	if err != nil {
		return nil, err
	}
	return &taskDir{rootDir: tempIO}, nil
}

func (e *taskDir) Cleanup() {
	os.RemoveAll(e.rootDir)
}

func (e *taskDir) WorkspacePath() string {
	return filepath.Join(e.rootDir, "work")
}

func (e *taskDir) PlannerPath() string {
	return filepath.Join(e.rootDir, "planner")
}

func (e *taskDir) ExecutorPath(attempt int) string {
	return filepath.Join(e.rootDir, fmt.Sprintf("executor-%d", attempt))
}

func (e *taskDir) ReviewPath(attempt int) string {
	return filepath.Join(e.rootDir, fmt.Sprintf("review-%d", attempt))
}
