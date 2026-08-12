package types

// Decision is the one agent-authored field the runtime branches on
type Review struct {
	Decision ReviewDecision `json:"decision"`

	// Summary is the reviewer's 1-2 sentence verdict, surfaced in run results.
	Summary string `json:"summary"`

	// Feedback is the reviewer's Markdown criticism:
	// Required when Decision is reject; welcome on accept too
	Feedback string `json:"feedback,omitempty"`
}

// ReviewDecision is the action the runtime takes next
type ReviewDecision string

const (
	// ReviewAccept ship the attempt's diff as the final output
	ReviewAccept ReviewDecision = "accept"

	// ReviewReject re-run the executor against the same Plan, passing the
	// feedback as additional context.
	ReviewReject ReviewDecision = "reject"
)

func NewReview(r Review) (Review, error) {
	if err := r.validate(); err != nil {
		return Review{}, err
	}
	return r, nil
}

func (r *Review) validate() error {
	var e errs
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
