package types

import (
	"github.com/HJyup/patchdock/internal/utils"
)

// ExecutionResult is the executor stage's output for one Plan attempt.
type ExecutionResult struct {
	ID     string `json:"id"`      // runtime-filled
	TaskID string `json:"task_id"` // runtime-filled
	PlanID string `json:"plan_id"` // runtime-filled

	// Status summarises the attempt. Everything else it wants to say goes in Notes
	Status ExecutionStatus `json:"status"`

	// Notes is the executor's Markdown account of what it did
	Notes string `json:"notes,omitempty"`
}

type ExecutionStatus string

const (
	// ExecutionSuccess the executor completed the plan
	ExecutionSuccess ExecutionStatus = "success"

	// ExecutionPartialSuccess part of the plan was completed; the reviewer
	// decides whether the resulting diff is worth accepting
	ExecutionPartialSuccess ExecutionStatus = "partial_success"

	// ExecutionFailed unrecoverable failure. The diff may be empty or partial
	ExecutionFailed ExecutionStatus = "failed"
)

func NewExecutionResult(x ExecutionResult) (ExecutionResult, error) {
	if x.ID == "" {
		x.ID = utils.NewID("exec")
	}
	if err := x.validate(); err != nil {
		return ExecutionResult{}, err
	}
	return x, nil
}

func (x *ExecutionResult) validate() error {
	var e errs
	e.required("execution_result.id", x.ID)
	e.required("execution_result.task_id", x.TaskID)
	e.required("execution_result.plan_id", x.PlanID)
	switch x.Status {
	case ExecutionSuccess, ExecutionPartialSuccess, ExecutionFailed:
	case "":
		e.addf("execution_result.status: empty")
	default:
		e.addf("execution_result.status: invalid value %q", x.Status)
	}
	return e.join()
}
