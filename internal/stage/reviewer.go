package stage

import (
	"context"
	"encoding/json"

	"github.com/HJyup/patchdock/internal/docker"
	"github.com/HJyup/patchdock/internal/types"
)

type ReviewerOpts struct {
	Image string
	Dir   string
	// WorkspaceDir, when set, is the target repository mounted where we can make any changes
	WorkspaceDir string
	AgentsDir    string
}

func RunReviewer(ctx context.Context, c *docker.Client, input ReviewerInput, exOpts ReviewerOpts) (types.Review, error) {
	var mounts []docker.Mount
	if exOpts.WorkspaceDir != "" {
		mounts = append(mounts, docker.Mount{Source: exOpts.WorkspaceDir, Target: WorkspaceTarget, ReadOnly: true})
	}

	raw, err := runStage(ctx, c, opts{
		image:      exOpts.Image,
		stage:      types.StageReviewer,
		taskID:     input.Plan.TaskID,
		dir:        exOpts.Dir,
		mounts:     mounts,
		agentsPath: exOpts.AgentsDir,
	}, input)
	if err != nil {
		return types.Review{}, err
	}

	var rev types.Review
	if err := json.Unmarshal(raw, &rev); err != nil {
		return types.Review{}, ErrOutputNotJSON{Err: err}
	}

	rev.TaskID = input.Plan.ID
	rev.ExecutionID = input.ExecutionResults[len(input.ExecutionResults)-1].ID

	res, err := types.NewReview(rev)
	if err != nil {
		return types.Review{}, ErrContractInvalid{Err: err}
	}

	return res, nil
}
