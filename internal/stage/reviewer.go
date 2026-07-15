package stage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/HJyup/patchdock/internal/docker"
	"github.com/HJyup/patchdock/internal/types"
)

type ReviewerOpts struct {
	Dir string
	// WorkspaceDir, when set, is the target repository mounted read-only
	// so the reviewer can inspect the executor's changes without editing them
	WorkspaceDir string

	AgentFile   string
	Attempt     int
	MaxAttempts int
}

func RunReviewer(ctx context.Context, c *docker.Client, input ReviewerInput, revOpts ReviewerOpts, agentOpts AgentOpts) (types.Review, error) {
	if len(input.ExecutionResults) == 0 {
		return types.Review{}, fmt.Errorf("reviewer requires at least one execution result")
	}

	var mounts []docker.Mount
	if revOpts.WorkspaceDir != "" {
		mounts = append(mounts, docker.Mount{Source: revOpts.WorkspaceDir, Target: WorkspaceTarget, ReadOnly: true})
	}

	raw, err := runStage(ctx, c, opts{
		image:       agentOpts.Image,
		stage:       types.StageReviewer,
		taskID:      input.Plan.TaskID,
		dir:         revOpts.Dir,
		mounts:      mounts,
		agentsPath:  agentOpts.AgentsDir,
		logger:      agentOpts.LogWriter,
		agentFile:   revOpts.AgentFile,
		timeout:     agentOpts.Timeout,
		maxTokens:   agentOpts.MaxTokens,
		attempt:     revOpts.Attempt,
		maxAttempts: revOpts.MaxAttempts,
	}, input)
	if err != nil {
		return types.Review{}, err
	}

	var rev types.Review
	if err := json.Unmarshal(raw, &rev); err != nil {
		return types.Review{}, ErrOutputNotJSON{Err: err}
	}

	rev.TaskID = input.Plan.TaskID
	rev.ExecutionID = input.ExecutionResults[len(input.ExecutionResults)-1].ID

	res, err := types.NewReview(rev)
	if err != nil {
		return types.Review{}, ErrContractInvalid{Err: err}
	}

	return res, nil
}
