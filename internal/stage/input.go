package stage

import (
	"io"
	"time"

	"github.com/HJyup/patchdock/internal/types"
)

type AgentOpts struct {
	Image     string
	AgentsDir string
	LogWriter io.Writer
	Timeout   time.Duration
	MaxTokens int
}

type PlannerInput struct {
	Task types.Task `json:"task"`
}

type ExecutorInput struct {
	Plan    types.Plan     `json:"plan"`
	Reviews []types.Review `json:"reviews"`
}

type ReviewerInput struct {
	Plan             types.Plan              `json:"plan"`
	ExecutionResults []types.ExecutionResult `json:"execution_results"`
	PreviousReviews  []types.Review          `json:"previous_reviews"`
}
