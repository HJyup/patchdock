package stage

import (
	"context"
	"encoding/json"
	"io"

	"github.com/HJyup/patchdock/internal/docker"
	"github.com/HJyup/patchdock/internal/types"
)

type PlannerOpts struct {
	Image     string
	Dir       string
	LogWriter io.Writer
	// RepoDir, when set, is the target repository mounted read-only at /repo
	// so the planner can explore the code it plans against.
	RepoDir   string
	AgentsDir string
}

func RunPlanner(ctx context.Context, c *docker.Client, input PlannerInput, plOpts PlannerOpts) (types.Plan, error) {
	var mounts []docker.Mount
	if plOpts.RepoDir != "" {
		mounts = append(mounts, docker.Mount{Source: plOpts.RepoDir, Target: RepoTarget, ReadOnly: true})
	}

	raw, err := runStage(ctx, c, opts{
		image:      plOpts.Image,
		stage:      types.StagePlanner,
		taskID:     input.Task.ID,
		dir:        plOpts.Dir,
		mounts:     mounts,
		agentsPath: plOpts.AgentsDir,
		logger:     plOpts.LogWriter,
	}, input)
	if err != nil {
		return types.Plan{}, err
	}

	var p types.Plan
	if err := json.Unmarshal(raw, &p); err != nil {
		return types.Plan{}, ErrOutputNotJSON{Err: err}
	}

	p.TaskID = input.Task.ID
	plan, err := types.NewPlan(p)
	if err != nil {
		return types.Plan{}, ErrContractInvalid{Err: err}
	}

	return plan, nil
}
