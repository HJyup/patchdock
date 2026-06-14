package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/HJyup/patchdock/internal/docker"
	"github.com/HJyup/patchdock/internal/stage"
	"github.com/HJyup/patchdock/internal/types"
)

type Pipeline struct {
	cli        *docker.Client
	image      string
	repoDir    string
	agentsDir  string
	maxRetries int
}

type Outcome struct {
	TaskID    string
	Plan      types.Plan
	Execution types.ExecutionResult
	Review    types.Review
	Attempts  int
	Accepted  bool
}

func NewPipeline(cli *docker.Client, image, repoDir, agentsDir string, maxRetries int) Pipeline {
	return Pipeline{
		cli:        cli,
		image:      image,
		repoDir:    repoDir,
		agentsDir:  agentsDir,
		maxRetries: maxRetries,
	}
}

func (p *Pipeline) Run(ctx context.Context, task types.Task) (*Outcome, error) {
	out := &Outcome{
		TaskID:   task.ID,
		Accepted: false,
	}

	if p.maxRetries < 0 {
		return out, fmt.Errorf("maximum amount of retries is incorrect: %v", p.maxRetries)
	}

	exists, err := p.cli.ImageExists(ctx, p.image)
	if err != nil {
		return out, err
	}

	if !exists {
		return out, fmt.Errorf("image %q not found — build it first:\n  docker build -t %s sdk/", p.image, p.repoDir)
	}

	tempIO, err := os.MkdirTemp("", "patchdock-io-*")
	if err != nil {
		return out, fmt.Errorf("create exchange parent: %w", err)
	}

	plan, err := stage.RunPlanner(ctx, p.cli, stage.PlannerInput{Task: task}, stage.PlannerOpts{
		Image:     p.image,
		Dir:       filepath.Join(tempIO, "planner"),
		RepoDir:   p.repoDir,
		AgentsDir: p.agentsDir,
	})
	if err != nil {
		return out, fmt.Errorf("planner stage: %w", err)
	}

	out.Plan = plan
	var execRes []types.ExecutionResult
	var revs []types.Review

	// 0 - first actual try (it doesn't count as a retry)
	for attempts := 0; attempts <= p.maxRetries; attempts++ {
		res, err := stage.RunExecutor(ctx, p.cli, stage.ExecutorInput{
			Plan:    plan,
			Reviews: revs,
		}, stage.ExecutorOpts{
			Image:        p.image,
			Dir:          filepath.Join(tempIO, fmt.Sprintf("executor-%v", attempts)),
			WorkspaceDir: "",
			AgentsDir:    p.agentsDir,
		})

		if err != nil {
			return out, fmt.Errorf("executor stage: %w", err)
		}

		execRes = append(execRes, res)
		out.Execution = res

		rev, err := stage.RunReviewer(ctx, p.cli, stage.ReviewerInput{
			Plan:             plan,
			ExecutionResults: execRes,
			PreviousReviews:  revs,
		}, stage.ReviewerOpts{
			Image:        p.image,
			Dir:          filepath.Join(tempIO, fmt.Sprintf("review-%v", attempts)),
			WorkspaceDir: "",
			AgentsDir:    p.agentsDir,
		})

		if err != nil {
			return out, fmt.Errorf("reviewer reviewer: %w", err)
		}

		revs = append(revs, rev)
		out.Review = rev
		out.Attempts = len(execRes)

		if rev.Decision == types.ReviewAccept {
			out.Accepted = true
			break
		}
	}

	return out, nil
}
