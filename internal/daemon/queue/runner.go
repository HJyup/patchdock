package queue

import (
	"context"

	"github.com/HJyup/patchdock/internal/types"
)

// RunSpec is everything a Runner needs to execute one run
type RunSpec struct {
	RunID string
	Repo  string
	Task  types.Task
}

type Outcome struct {
	Attempts int
	Accepted bool
	Branch   string
}

type Runner func(ctx context.Context, spec RunSpec, rep Reporter) (Outcome, error)
