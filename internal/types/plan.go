package types

import (
	"time"
)

// Plan is the planner stage's output: an immutable description of the work
// the executor should attempt for a single run.
type Plan struct {
	CreatedAt time.Time `json:"created_at"` // runtime-filled

	// Summary is the planner's 1-2 sentence account of the strategy,
	// surfaced in run results and status output.
	Summary string `json:"summary"`

	// Body is the full plan as markdown. Consumed by the executor and reviewer,
	// never parsed by the runtime.
	Body string `json:"body"`
}

func NewPlan(p Plan) (Plan, error) {
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now().UTC()
	}
	if err := p.validate(); err != nil {
		return Plan{}, err
	}
	return p, nil
}

func (p *Plan) validate() error {
	var e errs
	if p.CreatedAt.IsZero() {
		e.addf("plan.created_at: empty")
	}
	e.required("plan.summary", p.Summary)
	e.required("plan.body", p.Body)
	return e.join()
}
