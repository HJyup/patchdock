package pipeline

import (
	"context"
	"fmt"

	"github.com/HJyup/patchdock/internal/auditlog"
	"github.com/HJyup/patchdock/internal/config"
	"github.com/HJyup/patchdock/internal/stage"
	"github.com/HJyup/patchdock/internal/types"
	"github.com/HJyup/patchdock/internal/utils"
	"github.com/HJyup/patchdock/internal/workspace"
)

type Pipeline struct {
	cfg      config.Config
	runID    string
	repoDir  string
	runner   *stage.Runner
	logger   *auditlog.Logger
	reporter Reporter
}

type Outcome struct {
	Accepted bool
	Branch   string
	Patch    auditlog.PatchStat
}

func New(cfg config.Config, runID, repoDir string, runner *stage.Runner, logger *auditlog.Logger, reporter Reporter) *Pipeline {
	return &Pipeline{
		cfg:      cfg,
		runID:    runID,
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

	out = &Outcome{}
	history := newHistory()

	audit := newAuditRun(p.logger, p.runID, task)
	defer func() { audit.Finish(history, out, err) }()

	p.reporter.StageChange(types.StagePlanner, 0)
	plan, err := p.runner.RunPlanner(ctx, stage.PlannerRequest{
		Agent:       p.agentSpec(p.cfg.Stages[types.StagePlanner]),
		Input:       stage.PlannerInput{Task: task},
		ExchangeDir: dir.PlannerPath(),
		RepoDir:     p.repoDir,
	})

	if err != nil {
		audit.Failed(types.StagePlanner, err)
		return out, fmt.Errorf("planner stage: %w", err)
	}

	p.reporter.StageSummary(plan.Summary)
	audit.Planned(plan)

	wks, err := workspace.NewWorkspace(p.repoDir, dir.WorkspacePath())
	if err != nil {
		return out, fmt.Errorf("failed to create a workspace: %w", err)
	}

	for attempt := 1; attempt <= p.cfg.Retries.Max; attempt++ {
		p.reporter.StageChange(types.StageExecutor, attempt)
		res, err := p.runner.RunExecutor(ctx, stage.ExecutorRequest{
			Agent: p.agentSpec(p.cfg.Stages[types.StageExecutor]),
			Input: stage.ExecutorInput{
				Plan:    plan,
				Reviews: history.Reviews,
			},
			ExchangeDir:  dir.ExecutorPath(attempt),
			WorkspaceDir: wks.Dir,
			Attempt:      stage.Attempt{Number: attempt, Maximum: p.cfg.Retries.Max},
		})

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

		out.Patch = auditlog.StatPatch(diff)
		audit.Patched(out.Patch)

		p.reporter.StageChange(types.StageReviewer, attempt)

		rev, err := p.runner.RunReviewer(ctx, stage.ReviewerRequest{
			Agent: p.agentSpec(p.cfg.Stages[types.StageReviewer]),
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

		if err != nil {
			audit.Failed(types.StageReviewer, err)
			return out, fmt.Errorf("reviewer stage: %w", err)
		}

		p.reporter.StageSummary(rev.Summary)
		history.AddReview(rev)

		if rev.Decision == types.ReviewAccept {
			out.Accepted = true
			break
		}
	}

	if !out.Accepted {
		return out, nil
	}

	branch := p.branchName()
	if err := wks.Publish(ctx, branch, p.commitMessage(task, plan)); err != nil {
		audit.Failed(types.StageReviewer, err)
		return out, fmt.Errorf("publish %s: %w", branch, err)
	}
	out.Branch = branch
	audit.Published(branch)

	return out, nil
}

func (p *Pipeline) branchName() string {
	prefix := p.cfg.Git.BranchPrefix
	if prefix == "" {
		return p.runID
	}

	return prefix + "/" + p.runID
}

func (p *Pipeline) commitMessage(task types.Task, plan types.Plan) string {
	summary := utils.FirstLine(task.Description)
	if plan.Summary != "" {
		summary = utils.FirstLine(plan.Summary)
	}

	return fmt.Sprintf("%s\n\nPatchdock run %s", summary, p.runID)
}

func (p *Pipeline) agentSpec(agentFile string) stage.AgentSpec {
	return stage.AgentSpec{
		AgentFile: agentFile,
		Limits: stage.Limits{
			Timeout:   p.cfg.Container.Timeout.Duration(),
			MaxTokens: p.cfg.Container.TokenBudget,
		},
	}
}
