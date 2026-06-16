package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
)

type taskEnv struct {
	rootDir string
}

func newTaskEnv() (*taskEnv, error) {
	tempIO, err := os.MkdirTemp("", "patchdock-io-*")
	if err != nil {
		return nil, err
	}
	return &taskEnv{rootDir: tempIO}, nil
}

func (e *taskEnv) Cleanup() {
	os.RemoveAll(e.rootDir)
}

func (e *taskEnv) WorkspacePath() string {
	return filepath.Join(e.rootDir, "work")
}

func (e *taskEnv) PlannerPath() string {
	return filepath.Join(e.rootDir, "planner")
}

func (e *taskEnv) ExecutorPath(i int) string {
	return filepath.Join(e.rootDir, fmt.Sprintf("executor-%d", i))
}

func (e *taskEnv) ReviewPath(i int) string {
	return filepath.Join(e.rootDir, fmt.Sprintf("review-%d", i))
}
