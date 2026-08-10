package queue

import (
	"context"

	"github.com/HJyup/patchdock/internal/auditlog"
	"github.com/HJyup/patchdock/internal/types"
)

// RunSpec is everything a Runner needs to execute one run
type RunSpec struct {
	RunID string
	Repo  string
	Task  types.Task
}

type Outcome struct {
	Accepted bool
	Branch   string
	Patch    auditlog.PatchStat
}

type Runner func(ctx context.Context, spec RunSpec, rep Reporter) (Outcome, error)
