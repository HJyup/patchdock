package stage

import "github.com/HJyup/patchdock/internal/types"

type PlannerInput struct {
	Task types.Task `json:"task"`
}

type ExecutorInput struct {
	Plan    types.Plan     `json:"plan"`
	Reviews []types.Review `json:"review"`
}

type ReviewerInput struct {
	Plan             types.Plan              `json:"plan"`
	ExecutionResults []types.ExecutionResult `json:"execution_results"`
	PreviousReviews  []types.Review          `json:"previous_reviews"`
}
