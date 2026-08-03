package stage

import (
	"context"
	"fmt"

	"github.com/HJyup/patchdock/internal/docker"
	"github.com/HJyup/patchdock/internal/types"
)

type ReviewerInput struct {
	Plan  types.Plan `json:"plan"`
	Patch string     `json:"patch"`

	ExecutionResults []types.ExecutionResult `json:"execution_results"`
	PreviousReviews  []types.Review          `json:"previous_reviews"`
}

type ReviewerRequest struct {
	Agent        AgentSpec
	Input        ReviewerInput
	ExchangeDir  string
	WorkspaceDir string
	Attempt      Attempt
}

func (r *Runner) RunReviewer(ctx context.Context, req ReviewerRequest) (types.Review, error) {
	if len(req.Input.ExecutionResults) == 0 {
		return types.Review{}, fmt.Errorf("reviewer requires at least one execution result")
	}

	var mounts []docker.Mount
	if req.WorkspaceDir != "" {
		mounts = append(mounts, docker.Mount{Source: req.WorkspaceDir, Target: workspacePath, ReadOnly: true})
	}

	raw, err := r.runStage(ctx, req.Agent, runOptions{
		stage:       types.StageReviewer,
		taskID:      req.Input.Plan.TaskID,
		dir:         req.ExchangeDir,
		mounts:      mounts,
		attempt:     req.Attempt.Number,
		maxAttempts: req.Attempt.Maximum,
	}, req.Input)
	if err != nil {
		return types.Review{}, err
	}

	return decodeOutput(raw, func(r *types.Review) {
		r.TaskID = req.Input.Plan.TaskID
		r.ExecutionID = req.Input.ExecutionResults[len(req.Input.ExecutionResults)-1].ID
	}, types.NewReview)
}
