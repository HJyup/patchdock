package checks

import (
	"context"
	"time"

	"github.com/HJyup/patchdock/internal/config"
	"github.com/HJyup/patchdock/internal/docker"
)

type CheckResult struct {
	Name       string `json:"name"`
	Command    string `json:"command"`
	ExitCode   int64  `json:"exit_code"`
	Passed     bool   `json:"passed"`
	Output     string `json:"output"`
	DurationMS int64  `json:"duration_ms"`
}

type Runner struct {
	cli                 *docker.Client
	image, workspaceDir string
	timeout             time.Duration
}

func (r *Runner) Run(ctx context.Context, cmds []config.Run) ([]CheckResult, error)
