package pipeline

import (
	"context"
	"fmt"

	"github.com/HJyup/patchdock/internal/docker"
	"github.com/HJyup/patchdock/internal/stage"
	"github.com/HJyup/patchdock/internal/types"
	"github.com/HJyup/patchdock/internal/workspace"
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
	err := p.validateEnv(ctx)
	if err != nil {
		return nil, err
	}

	env, err := newTaskEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize task environment: %w", err)
	}
	defer env.Cleanup()

	// Base outcome which will be populated after each stage
	out := &Outcome{
		TaskID:   task.ID,
		Accepted: false,
	}

	plan, err := stage.RunPlanner(ctx, p.cli, stage.PlannerInput{Task: task}, stage.PlannerOpts{
		Image:     p.image,
		Dir:       env.PlannerPath(),
		RepoDir:   p.repoDir,
		AgentsDir: p.agentsDir,
	})
	if err != nil {
		return out, fmt.Errorf("planner stage: %w", err)
	}

	out.Plan = plan
	history := newHistory()

	wks, err := workspace.NewWorkspace(p.repoDir, env.WorkspacePath())
	if err != nil {
		return out, fmt.Errorf("failed to create a workspace: %w", err)
	}

	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		res, err := stage.RunExecutor(ctx, p.cli, stage.ExecutorInput{
			Plan:    plan,
			Reviews: history.Reviews,
		}, stage.ExecutorOpts{
			Image:        p.image,
			Dir:          env.ExecutorPath(attempt),
			WorkspaceDir: wks.Dir,
			AgentsDir:    p.agentsDir,
		})
		if err != nil {
			return out, fmt.Errorf("executor stage: %w", err)
		}

		diff, err := wks.Diff(ctx)
		if err != nil {
			return out, fmt.Errorf("executor stage (failed computing diffs): %w", err)
		}
		res.Patch = diff

		history.AddExecution(res)
		out.Execution = res

		rev, err := stage.RunReviewer(ctx, p.cli, stage.ReviewerInput{
			Plan:             plan,
			ExecutionResults: history.Executions,
			PreviousReviews:  history.Reviews,
		}, stage.ReviewerOpts{
			Image:        p.image,
			Dir:          env.ReviewPath(attempt),
			WorkspaceDir: wks.Dir,
			AgentsDir:    p.agentsDir,
		})
		if err != nil {
			return out, fmt.Errorf("reviewer reviewer: %w", err)
		}

		history.AddReview(rev)
		out.Review = rev
		out.Attempts = len(history.Executions)

		if rev.Decision == types.ReviewAccept {
			out.Accepted = true
			break
		}
	}

	return out, nil
}

func (p *Pipeline) validateEnv(ctx context.Context) error {
	if p.maxRetries < 0 {
		return fmt.Errorf("maximum amount of retries is incorrect: %v", p.maxRetries)
	}

	exists, err := p.cli.ImageExists(ctx, p.image)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("image %q not found — build it first:\n  docker build -t %s sdk/", p.image, p.repoDir)
	}

	return nil
}
