package stage

import (
	"context"
	"encoding/json"

	"github.com/HJyup/patchdock/internal/docker"
	"github.com/HJyup/patchdock/internal/types"
)

type ExecutorInput struct {
	Plan    types.Plan     `json:"plan"`
	Reviews []types.Review `json:"reviews"`
}

type ExecutorRequest struct {
	Spec         StageSpec
	Input        ExecutorInput
	ExchangeDir  string
	WorkspaceDir string
	Attempt      Attempt
}

func (r *Runner) RunExecutor(ctx context.Context, req ExecutorRequest) (types.ExecutionResult, error) {
	var mounts []docker.Mount
	if req.WorkspaceDir != "" {
		mounts = append(mounts, docker.Mount{Source: req.WorkspaceDir, Target: WorkspaceTarget, ReadOnly: false})
	}

	raw, err := r.runStage(ctx, req.Spec, runOptions{
		stage:       types.StageExecutor,
		taskID:      req.Input.Plan.TaskID,
		dir:         req.ExchangeDir,
		mounts:      mounts,
		attempt:     req.Attempt.Number,
		maxAttempts: req.Attempt.Maximum,
	}, req.Input)
	if err != nil {
		return types.ExecutionResult{}, err
	}

	var ex types.ExecutionResult
	if err := json.Unmarshal(raw, &ex); err != nil {
		return types.ExecutionResult{}, ErrOutputNotJSON{Err: err}
	}

	ex.TaskID = req.Input.Plan.TaskID
	ex.PlanID = req.Input.Plan.ID

	res, err := types.NewExecutionResult(ex)
	if err != nil {
		return types.ExecutionResult{}, ErrContractInvalid{Err: err}
	}

	return res, nil
}
