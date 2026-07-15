package types

import "github.com/HJyup/patchdock/internal/id"

// ExecutionResult is the executor stage's output for one Plan attempt.
type ExecutionResult struct {
	ID     string `json:"id"`      // runtime-filled
	TaskID string `json:"task_id"` // runtime-filled
	PlanID string `json:"plan_id"` // runtime-filled

	// Status summarises the attempt. Everything else it wants to say goes in Notes
	Status ExecutionStatus `json:"status"`

	// Patch is the unified diff against the base commit, extracted from the
	// workspace by the runtime after the container exits
	Patch string `json:"patch,omitempty"`

	// Notes is the executor's markdown account of what it did, what worked,
	// and what didn't
	Notes string `json:"notes,omitempty"`
}

// ExecutionStatus summarizes the outcome of an execution attempt.
type ExecutionStatus string

const (
	// ExecutionSuccess the executor completed the plan; Patch is the proposed change.
	ExecutionSuccess ExecutionStatus = "success"

	// ExecutionPartialSuccess part of the plan was completed. Patch reflects
	// what was done; the reviewer decides whether to accept.
	ExecutionPartialSuccess ExecutionStatus = "partial_success"

	// ExecutionFailed unrecoverable failure. Patch may be empty or partial.
	ExecutionFailed ExecutionStatus = "failed"
)

func NewExecutionResult(x ExecutionResult) (ExecutionResult, error) {
	if x.ID == "" {
		x.ID = id.New("exec")
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
