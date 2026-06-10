package contracts

import "time"

// Plan is the planner stage's output: an ordered, immutable description of
// the work the executor should attempt for a single task attempt.
type Plan struct {
	ID string `json:"id" validate:"required"`

	// TaskID groups all artifacts (plans, executions, reviews) for one task run.
	TaskID    TaskID    `json:"task_id" validate:"required"`
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
	Assumptions []string   `json:"assumptions,omitempty"`
	TokensUsed  TokenUsage `json:"tokens_used"`
}

// Step is one unit of work in a Plan. The executor produces one StepResult
// per Step, keyed by Step.ID.
type Step struct {
	ID          string `json:"id" validate:"required"`
	Description string `json:"description" validate:"required"`

	// Rationale is why this step is necessary.
	// Optional: a step whose description carries its own justification may
	// leave it empty.
	Rationale string `json:"rationale"`

	// FilesToModify is the planner's intent. Executor may modify additional
	// files but the reviewer can flag deviations. (saves tokens)
	FilesToModify []string `json:"files_to_modify,omitempty"`
}

func (s *Step) Validate() error { return validateStruct(s, "step") }

// NewPlan completes a caller-assembled Plan and validates it. A zero ID and
// CreatedAt are generated; values already set are kept, so fixtures and
// tests can pin them for determinism. This is the only sanctioned way to
// produce a Plan outside decoding one at a stage boundary.
func NewPlan(p Plan) (Plan, error) {
	if p.ID == "" {
		p.ID = newID("plan")
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now().UTC()
	}
	if err := p.Validate(); err != nil {
		return Plan{}, err
	}
	return p, nil
}

func (p *Plan) Validate() error { return validateStruct(p, "plan") }
