package types

import (
	"time"

	"github.com/HJyup/patchdock/internal/id"
)

// Plan is the planner stage's output: an ordered, immutable description of
// the work the executor should attempt for a single task attempt.
type Plan struct {
	ID        string    `json:"id" validate:"required"`
	TaskID    string    `json:"task_id" validate:"required"`
	CreatedAt time.Time `json:"created_at" validate:"required"`

	// Approach is the planner's 1-3 sentence summary of the overall strategy.
	Approach string `json:"approach" validate:"required"`

	// AcceptanceCriteria is the planner's definition of "done" — what would
	// have to be true for this task to be considered complete. Used by the
	// executor as a self-check and by the reviewer as the evaluation bar.
	// Required (non-empty).
	AcceptanceCriteria []string `json:"acceptance_criteria" validate:"min=1,dive,required"`
	Steps              []Step   `json:"steps" validate:"min=1,dive"`

	// Context captures facts the planner discovered while exploring the repo
	// that the executor would otherwise re-discover. Saves tokens and reduces drift
	// between planner and executor understanding.
	Context []string `json:"context,omitempty"`

	// Assumptions the planner made that may not hold at execution time.
	Assumptions []string `json:"assumptions,omitempty"`
}

// Step is one unit of work in a Plan. The executor produces one StepResult
// per Step, keyed by Step.ID.
type Step struct {
	ID          string `json:"id" validate:"required"`
	Description string `json:"description" validate:"required"`

	// Rationale is why this step is necessary.
	// Optional: a step whose description carries its own justification may
	// leave it empty.
	Rationale string `json:"rationale,omitempty"`

	// FilesToModify is the planner's intent.
	// Executor may modify additional files
	FilesToModify []string `json:"files_to_modify,omitempty"`
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
	return validateStruct(p, "plan")
}
