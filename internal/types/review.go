package types

import "github.com/HJyup/patchdock/internal/utils"

// Review is the reviewer stage's output for one ExecutionResult.
//
// When Decision is Reject, the payload must carry enough structured info
// for the executor's next attempt to be productive.
type Review struct {
	ID          string         `json:"id" validate:"required"`
	TaskID      string         `json:"task_id" validate:"required"`
	ExecutionID string         `json:"execution_id" validate:"required"`
	Decision    ReviewDecision `json:"decision" validate:"required,oneof=accept reject"`

	// Issues found. Required (non-empty) when Decision is Reject.
	// Empty when Decision is Accept.
	Issues []ReviewIssue `json:"issues,omitempty" validate:"dive"`

	Summary string `json:"summary" validate:"required"`
}

// ReviewDecision is the action the orchestrator should take next.
//
// By design the planner runs at most once per task. If the reviewer rejects,
// the orchestrator re-runs the executor against the same Plan with the
// issues as additional context.
type ReviewDecision string

const (
	// ReviewAccept Ship the ExecutionResult.Patch as the final output.
	ReviewAccept ReviewDecision = "accept"

	// ReviewReject re-run the executor against the same Plan, passing the
	// issues as additional context.
	ReviewReject ReviewDecision = "reject"
)

// ReviewIssue is a structured criticism the executor or planner can act on.
type ReviewIssue struct {
	// Severity scales with how blocking this is.
	Severity  IssueSeverity `json:"severity" validate:"required,oneof=blocker major minor"`
	Message   string        `json:"message" validate:"required"`
	StepID    string        `json:"step_id,omitempty"`
	FilePath  string        `json:"file_path,omitempty"`
	LineRange string        `json:"line_range,omitempty"`

	// Suggestion is what the reviewer thinks should happen next. Optional but
	// strongly encouraged for blocker/major severities.
	Suggestion string `json:"suggestion,omitempty"`
}

// IssueSeverity grades how blocking an issue is.
type IssueSeverity string

const (
	// SeverityBlocker must be addressed before the change can be accepted.
	SeverityBlocker IssueSeverity = "blocker"

	// SeverityMajor should be addressed; reviewer may still reject if ignored.
	SeverityMajor IssueSeverity = "major"

	// SeverityMinor nice-to-have; reviewer will accept without it.
	SeverityMinor IssueSeverity = "minor"
)

func NewReview(r Review) (Review, error) {
	if r.ID == "" {
		r.ID = utils.NewID("review")
	}
	if err := r.validate(); err != nil {
		return Review{}, err
	}
	return r, nil
}

func (r *Review) validate() error {
	return validateStruct(r, "review")
}
