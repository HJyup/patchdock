package queue

import (
	"context"

	"github.com/HJyup/patchdock/internal/auditlog"
	"github.com/HJyup/patchdock/internal/types"
)

// RunSpec declares information that needed for one run in a queue
type RunSpec struct {
	RunID string
	Repo  string
	Task  types.Task
}

// Outcome is what a finished Runner returns back
type Outcome struct {
	Accepted bool
	Branch   string
	Patch    auditlog.PatchStat
}

// Runner executes one run to completion, reporting stage progress through reporter
type Runner func(ctx context.Context, spec RunSpec, rep Reporter) (Outcome, error)
