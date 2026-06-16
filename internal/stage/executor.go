package stage

import (
	"context"
	"encoding/json"
	"io"

	"github.com/HJyup/patchdock/internal/docker"
	"github.com/HJyup/patchdock/internal/types"
)

type ExecutorOpts struct {
	Image     string
	Dir       string
	LogWriter io.Writer
	// WorkspaceDir, when set, is the target repository mounted where we can make any changes
	WorkspaceDir string
	AgentsDir    string
}

func RunExecutor(ctx context.Context, c *docker.Client, input ExecutorInput, exOpts ExecutorOpts) (types.ExecutionResult, error) {
	var mounts []docker.Mount
	if exOpts.WorkspaceDir != "" {
		mounts = append(mounts, docker.Mount{Source: exOpts.WorkspaceDir, Target: WorkspaceTarget, ReadOnly: false})
	}

	raw, err := runStage(ctx, c, opts{
		image:      exOpts.Image,
		stage:      types.StageExecutor,
		taskID:     input.Plan.TaskID,
		dir:        exOpts.Dir,
		mounts:     mounts,
		agentsPath: exOpts.AgentsDir,
		logger:     exOpts.LogWriter,
	}, input)
	if err != nil {
		return types.ExecutionResult{}, err
	}

	var ex types.ExecutionResult
	if err := json.Unmarshal(raw, &ex); err != nil {
		return types.ExecutionResult{}, ErrOutputNotJSON{Err: err}
	}

	ex.TaskID = input.Plan.ID
	ex.PlanID = input.Plan.ID

	res, err := types.NewExecutionResult(ex)
	if err != nil {
		return types.ExecutionResult{}, ErrContractInvalid{Err: err}
	}

	return res, nil
}
