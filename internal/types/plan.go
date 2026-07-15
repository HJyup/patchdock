package types

import (
	"time"

	"github.com/HJyup/patchdock/internal/id"
)

// Plan is the planner stage's output: an immutable description of the work
// the executor should attempt for a single task.
type Plan struct {
	ID        string    `json:"id"`         // runtime-filled
	TaskID    string    `json:"task_id"`    // runtime-filled
	CreatedAt time.Time `json:"created_at"` // runtime-filled

	// Summary is the planner's 1-2 sentence account of the strategy,
	// surfaced in run results and status output.
	Summary string `json:"summary"`

	// Body is the full plan as markdown. Consumed by the executor and reviewer,
	// never parsed by the runtime.
	Body string `json:"body"`
}

func NewPlan(p Plan) (Plan, error) {
	if p.ID == "" {
		p.ID = id.New("plan")
	}
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
	e.required("plan.id", p.ID)
	e.required("plan.task_id", p.TaskID)
	if p.CreatedAt.IsZero() {
		e.addf("plan.created_at: empty")
	}
	e.required("plan.summary", p.Summary)
	e.required("plan.body", p.Body)
	return e.join()
}
