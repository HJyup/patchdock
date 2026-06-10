package contracts

// ExecutionResult is the executor stage's output for one Plan attempt.
//
// One ExecutionResult per executor invocation. A retry (after a reject)
// produces a fresh ExecutionResult; the old one is preserved for audit.
type ExecutionResult struct {
	ID string `json:"id" validate:"required"`

	TaskID TaskID          `json:"task_id" validate:"required"`
	PlanID string          `json:"plan_id" validate:"required"`
	Status ExecutionStatus `json:"status" validate:"required,oneof=success partial_success failed"`

	// Patch is the unified diff against the target repo, as the executor sees it.
	// Empty when Status is Failed before any modification happened.
	Patch string `json:"patch,omitempty"`

	// StepResults are an in-order prefix of Plan.Steps, keyed by Step.ID.
	// Missing trailing entries mean "executor didn't get this far."
	StepResults []StepResult `json:"step_results" validate:"dive"`

	// Errors records things that went wrong during execution. (Not opinions about the output)
	Errors     []ExecutionError `json:"errors,omitempty" validate:"dive"`
	TokensUsed TokenUsage       `json:"tokens_used"`
}

// ExecutionStatus summarizes the outcome of an execution attempt.
type ExecutionStatus string

const (
	// ExecutionSuccess every Step completed; Patch is the proposed change.
	ExecutionSuccess ExecutionStatus = "success"

	// ExecutionPartialSuccess some Steps completed, others did not.
	// Patch reflects what was completed; reviewer decides whether to accept.
	ExecutionPartialSuccess ExecutionStatus = "partial_success"

	// ExecutionFailed unrecoverable failure. Patch may be empty or partial.
	ExecutionFailed ExecutionStatus = "failed"
)

// StepResult is the executor's record of one Step.
type StepResult struct {
	StepID string          `json:"step_id" validate:"required"`
	Status ExecutionStatus `json:"status" validate:"required,oneof=success partial_success failed"`

	// Notes is the executor's optional human-readable commentary on this step
	Notes string `json:"notes,omitempty"`
}

// Validate reports every broken invariant at once, each error naming the
// offending field.
func (s *StepResult) Validate() error {
	return validateStruct(s, "step_result")
}

// ExecutionError is a harness-level failure during execution.
type ExecutionError struct {
	// StepID, if set, scopes the error to one Step. Empty means whole-execution.
	StepID  string `json:"step_id,omitempty"`
	Message string `json:"message" validate:"required"`
}

func (e *ExecutionError) Validate() error {
	return validateStruct(e, "execution_error")
}

// NewExecutionResult completes a caller-assembled ExecutionResult and
// validates it. A zero ID is generated; a set ID is kept for determinism.
func NewExecutionResult(e ExecutionResult) (ExecutionResult, error) {
	if e.ID == "" {
		e.ID = newID("exec")
	}
	if err := e.Validate(); err != nil {
		return ExecutionResult{}, err
	}
	return e, nil
}

func (e *ExecutionResult) Validate() error {
	return validateStruct(e, "execution_result")
}
