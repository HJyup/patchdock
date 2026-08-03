package pipeline

import (
	"context"
	"fmt"

	"github.com/HJyup/patchdock/internal/auditlog"
	"github.com/HJyup/patchdock/internal/config"
	"github.com/HJyup/patchdock/internal/stage"
	"github.com/HJyup/patchdock/internal/types"
	"github.com/HJyup/patchdock/internal/workspace"
)

type Pipeline struct {
	cfg      config.Config
	repoDir  string
	runner   *stage.Runner
	logger   *auditlog.Logger
	reporter Reporter
}

type Outcome struct {
	TaskID   string
	Attempts int
	Accepted bool
}

func New(cfg config.Config, repoDir string, runner *stage.Runner, logger *auditlog.Logger, reporter Reporter) *Pipeline {
	if reporter == nil {
		reporter = stubReporter{}
	}

	return &Pipeline{
		cfg:      cfg,
		repoDir:  repoDir,
		reporter: reporter,
		runner:   runner,
		logger:   logger,
	}
}

func (p *Pipeline) Run(ctx context.Context, task types.Task) (out *Outcome, err error) {
	dir, err := newTemporaryDir()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize task environment: %w", err)
	}
	defer dir.Cleanup()

	out = &Outcome{TaskID: task.ID}
	history := newHistory()

	audit := newAuditRun(p.logger, task)
	defer func() { audit.Finish(history, out, err) }()

	p.reporter.StageStarted(types.StagePlanner, 0)

	plan, err := p.runner.RunPlanner(ctx, stage.PlannerRequest{
		Spec:        p.stageSpec(p.cfg.Stages[types.StagePlanner]),
		Input:       stage.PlannerInput{Task: task},
		ExchangeDir: dir.PlannerPath(),
		RepoDir:     p.repoDir,
	})

	p.reporter.StageFinished(types.StagePlanner, 0, "", err)
	if err != nil {
		audit.Failed(types.StagePlanner, err)
		return out, fmt.Errorf("planner stage: %w", err)
	}

	audit.Planned(plan)

	wks, err := workspace.NewWorkspace(p.repoDir, dir.WorkspacePath())
	if err != nil {
		return out, fmt.Errorf("failed to create a workspace: %w", err)
	}

	for attempt := 1; attempt <= p.cfg.Retries.Max; attempt++ {
		p.reporter.StageStarted(types.StageExecutor, attempt)

		res, err := p.runner.RunExecutor(ctx, stage.ExecutorRequest{
			Spec: p.stageSpec(p.cfg.Stages[types.StageExecutor]),
			Input: stage.ExecutorInput{
				Plan:    plan,
				Reviews: history.Reviews,
			},
			ExchangeDir:  dir.ExecutorPath(attempt),
			WorkspaceDir: wks.Dir,
			Attempt:      stage.Attempt{Number: attempt, Maximum: p.cfg.Retries.Max},
		})

		p.reporter.StageFinished(types.StageExecutor, attempt, string(res.Status), err)
		if err != nil {
			audit.Failed(types.StageExecutor, err)
			return out, fmt.Errorf("executor stage: %w", err)
		}

		history.AddExecution(res)

		diff, err := wks.Diff(ctx)
		if err != nil {
			audit.Failed(types.StageExecutor, err)
			return out, fmt.Errorf("executor stage (failed computing diffs): %w", err)
		}

		if err := audit.Patched(diff); err != nil {
			return out, fmt.Errorf("write workspace patch: %w", err)
		}

		p.reporter.StageStarted(types.StageReviewer, attempt)

		rev, err := p.runner.RunReviewer(ctx, stage.ReviewerRequest{
			Spec: p.stageSpec(p.cfg.Stages[types.StageReviewer]),
			Input: stage.ReviewerInput{
				Plan:             plan,
				Patch:            diff,
				ExecutionResults: history.Executions,
				PreviousReviews:  history.Reviews,
			},
			ExchangeDir:  dir.ReviewPath(attempt),
			WorkspaceDir: wks.Dir,
			Attempt:      stage.Attempt{Number: attempt, Maximum: p.cfg.Retries.Max},
		})

		p.reporter.StageFinished(types.StageReviewer, attempt, string(rev.Decision), err)
		if err != nil {
			audit.Failed(types.StageReviewer, err)
			return out, fmt.Errorf("reviewer stage: %w", err)
		}

		history.AddReview(rev)
		out.Attempts = attempt

		if rev.Decision == types.ReviewAccept {
			out.Accepted = true
			break
		}
	}

	return out, nil
}

func (p *Pipeline) stageSpec(agentFile string) stage.Spec {
	return stage.Spec{
		AgentFile: agentFile,
		Limits: stage.Limits{
			Timeout:   p.cfg.Container.Timeout.Duration(),
			MaxTokens: p.cfg.Container.TokenBudget,
		},
	}
}
