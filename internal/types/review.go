package types

import "github.com/HJyup/patchdock/internal/id"

// Review is the reviewer stage's output for one ExecutionResult.
// Decision is the one agent-authored field the runtime branches on
type Review struct {
	ID          string `json:"id"`           // runtime-filled
	TaskID      string `json:"task_id"`      // runtime-filled
	ExecutionID string `json:"execution_id"` // runtime-filled

	Decision ReviewDecision `json:"decision"`

	// Summary is the reviewer's 1-2 sentence verdict, surfaced in run results.
	Summary string `json:"summary"`

	// Feedback is the reviewer's markdown criticism:
	// Required when Decision is reject; welcome on accept too
	Feedback string `json:"feedback,omitempty"`
}

// ReviewDecision is the action the orchestrator should take next.

type ReviewDecision string

const (
	// ReviewAccept ship the ExecutionResult.Patch as the final output.
	ReviewAccept ReviewDecision = "accept"

	// ReviewReject re-run the executor against the same Plan, passing the
	// feedback as additional context.
	ReviewReject ReviewDecision = "reject"
)

func NewReview(r Review) (Review, error) {
	if r.ID == "" {
		r.ID = id.New("review")
	}
	if err := r.validate(); err != nil {
		return Review{}, err
	}
	return r, nil
}

func (r *Review) validate() error {
	var e errs
	e.required("review.id", r.ID)
	e.required("review.task_id", r.TaskID)
	e.required("review.execution_id", r.ExecutionID)
	switch r.Decision {
	case ReviewAccept, ReviewReject:
	case "":
		e.addf("review.decision: empty")
	default:
		e.addf("review.decision: invalid value %q", r.Decision)
	}
	e.required("review.summary", r.Summary)
	if r.Decision == ReviewReject && r.Feedback == "" {
		e.addf("review.feedback: required when decision is reject")
	}
	return e.join()
}
