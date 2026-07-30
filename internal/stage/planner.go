package stage

import (
	"context"
	"encoding/json"

	"github.com/HJyup/patchdock/internal/docker"
	"github.com/HJyup/patchdock/internal/types"
)

type PlannerInput struct {
	Task types.Task `json:"task"`
}

type PlannerRequest struct {
	Spec        Spec
	Input       PlannerInput
	ExchangeDir string
	RepoDir     string
	Attempt     Attempt
}

func (r *Runner) RunPlanner(ctx context.Context, req PlannerRequest) (types.Plan, error) {
	var mounts []docker.Mount
	if req.RepoDir != "" {
		mounts = append(mounts, docker.Mount{Source: req.RepoDir, Target: repoPath, ReadOnly: true})
	}

	raw, err := r.runStage(ctx, req.Spec, runOptions{
		stage:       types.StagePlanner,
		taskID:      req.Input.Task.ID,
		dir:         req.ExchangeDir,
		mounts:      mounts,
		attempt:     req.Attempt.Number,
		maxAttempts: req.Attempt.Maximum,
	}, req.Input)
	if err != nil {
		return types.Plan{}, err
	}

	var p types.Plan
	if err := json.Unmarshal(raw, &p); err != nil {
		return types.Plan{}, ErrOutput{Reason: reasonNotJSON, Err: err, Raw: raw}
	}

	p.TaskID = req.Input.Task.ID
	plan, err := types.NewPlan(p)
	if err != nil {
		return types.Plan{}, ErrOutput{Reason: reasonContract, Err: err, Raw: raw}
	}

	return plan, nil
}
