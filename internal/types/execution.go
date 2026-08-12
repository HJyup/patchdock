package types

type ExecutionResult struct {
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
	if err := x.validate(); err != nil {
		return ExecutionResult{}, err
	}
	return x, nil
}

func (x *ExecutionResult) validate() error {
	var e errs
	switch x.Status {
	case ExecutionSuccess, ExecutionPartialSuccess, ExecutionFailed:
	case "":
		e.addf("execution_result.status: empty")
	default:
		e.addf("execution_result.status: invalid value %q", x.Status)
	}
	return e.join()
}
