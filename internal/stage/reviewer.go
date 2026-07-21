package stage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/HJyup/patchdock/internal/docker"
	"github.com/HJyup/patchdock/internal/types"
)

type ReviewerInput struct {
	Plan             types.Plan              `json:"plan"`
	ExecutionResults []types.ExecutionResult `json:"execution_results"`
	PreviousReviews  []types.Review          `json:"previous_reviews"`
}

type ReviewerRequest struct {
	Spec         StageSpec
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
		mounts = append(mounts, docker.Mount{Source: req.WorkspaceDir, Target: WorkspaceTarget, ReadOnly: true})
	}

	raw, err := r.runStage(ctx, req.Spec, runOptions{
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

	var rev types.Review
	if err := json.Unmarshal(raw, &rev); err != nil {
		return types.Review{}, ErrOutputNotJSON{Err: err}
	}

	rev.TaskID = req.Input.Plan.TaskID
	rev.ExecutionID = req.Input.ExecutionResults[len(req.Input.ExecutionResults)-1].ID

	res, err := types.NewReview(rev)
	if err != nil {
		return types.Review{}, ErrContractInvalid{Err: err}
	}

	return res, nil
}
