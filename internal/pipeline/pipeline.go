package pipeline

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/HJyup/patchdock/internal/auditlog"
	"github.com/HJyup/patchdock/internal/auth"
	"github.com/HJyup/patchdock/internal/config"
	"github.com/HJyup/patchdock/internal/docker"
	"github.com/HJyup/patchdock/internal/stage"
	"github.com/HJyup/patchdock/internal/types"
	"github.com/HJyup/patchdock/internal/workspace"
)

type Pipeline struct {
	cli          *docker.Client
	cfg          config.Config
	imageTag     string
	repoDir      string
	patchdockDir string
	maxAttempts  int
	reporter     Reporter
}

type Outcome struct {
	TaskID   string
	Attempts int
	Accepted bool
}

func NewPipeline(cli *docker.Client, cfg config.Config, imageTag, repoDir, patchdockDir string, reporter Reporter) *Pipeline {
	if reporter == nil {
		reporter = stubReporter{}
	}

	return &Pipeline{
		cli:          cli,
		cfg:          cfg,
		imageTag:     imageTag,
		repoDir:      repoDir,
		patchdockDir: patchdockDir,
		maxAttempts:  cfg.Retries.Max + 1,
		reporter:     reporter,
	}
}

func (p *Pipeline) Run(ctx context.Context, task types.Task) (out *Outcome, err error) {
	dir, err := newTemporaryDir()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize task environment: %w", err)
	}
	defer dir.Cleanup()

	runID := fmt.Sprintf("%s-%s", task.ID, time.Now().Format("20060102-150405"))
	logDir := filepath.Join(p.patchdockDir, "logs", runID)

	logger, err := auditlog.New(logDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize audit logger: %w", err)
	}
	defer logger.Close()

	out = &Outcome{TaskID: task.ID}
	history := newHistory()

	rec := &auditlog.Record{RunID: runID, Task: task, StartedAt: time.Now()}
	failed, rawKept := "setup", false
	defer func() {
		rec.Attempts = history.auditAttempts()
		rec.Accepted = out.Accepted
		rec.Duration = time.Since(rec.StartedAt).Round(time.Second).String()
		if err != nil {
			rec.Failure = &auditlog.Failure{Stage: failed, Message: err.Error(), RawOutput: rawKept}
		}
		if writeErr := logger.WriteRun(rec); writeErr != nil {
			fmt.Fprintf(logger, "audit: failed to write run record: %v\n", writeErr)
		}
	}()

	keepRawOutput := func(stageErr error) {
		raw := stage.RawOutput(stageErr)
		if len(raw) == 0 {
			return
		}
		if writeErr := logger.WriteFailedOutput(raw); writeErr != nil {
			fmt.Fprintf(logger, "audit: failed to archive raw output: %v\n", writeErr)
			return
		}
		rawKept = true
	}

	cred, err := auth.LoadCodex(p.cfg.Codex)
	if err != nil {
		return out, fmt.Errorf("load Codex credentials: %w", err)
	}

	stages := stage.NewRunner(p.cli, stage.RunnerOptions{
		ImageTag:     p.imageTag,
		PatchdockDir: p.patchdockDir,
		LogWriter:    logger,
		Credentials:  cred,
		OnActivity:   p.reporter.StageActivity,
	})

	failed = string(types.StagePlanner)
	p.reporter.StageStarted(types.StagePlanner, 0)
	plan, err := stages.RunPlanner(ctx, stage.PlannerRequest{
		Spec:        p.stageSpec(p.cfg.Stages[types.StagePlanner]),
		Input:       stage.PlannerInput{Task: task},
		ExchangeDir: dir.PlannerPath(),
		RepoDir:     p.repoDir,
	})
	p.reporter.StageFinished(types.StagePlanner, 0, "", err)
	if err != nil {
		keepRawOutput(err)
		return out, fmt.Errorf("planner stage: %w", err)
	}

	rec.Plan = plan

	wks, err := workspace.NewWorkspace(p.repoDir, dir.WorkspacePath())
	if err != nil {
		return out, fmt.Errorf("failed to create a workspace: %w", err)
	}

	for attempt := 1; attempt <= p.maxAttempts; attempt++ {
		failed = string(types.StageExecutor)
		p.reporter.StageStarted(types.StageExecutor, attempt)
		res, err := stages.RunExecutor(ctx, stage.ExecutorRequest{
			Spec: p.stageSpec(p.cfg.Stages[types.StageExecutor]),
			Input: stage.ExecutorInput{
				Plan:    plan,
				Reviews: history.Reviews,
			},
			ExchangeDir:  dir.ExecutorPath(attempt),
			WorkspaceDir: wks.Dir,
			Attempt:      stage.Attempt{Number: attempt, Maximum: p.maxAttempts},
		})
		p.reporter.StageFinished(types.StageExecutor, attempt, string(res.Status), err)
		if err != nil {
			keepRawOutput(err)
			return out, fmt.Errorf("executor stage: %w", err)
		}

		history.AddExecution(res)

		diff, err := wks.Diff(ctx)
		if err != nil {
			return out, fmt.Errorf("executor stage (failed computing diffs): %w", err)
		}

		if err := logger.WritePatch(diff); err != nil {
			return out, fmt.Errorf("write workspace patch: %w", err)
		}
		rec.Patch = auditlog.StatPatch(diff)

		failed = string(types.StageReviewer)
		p.reporter.StageStarted(types.StageReviewer, attempt)
		rev, err := stages.RunReviewer(ctx, stage.ReviewerRequest{
			Spec: p.stageSpec(p.cfg.Stages[types.StageReviewer]),
			Input: stage.ReviewerInput{
				Plan:             plan,
				Patch:            diff,
				ExecutionResults: history.Executions,
				PreviousReviews:  history.Reviews,
			},
			ExchangeDir:  dir.ReviewPath(attempt),
			WorkspaceDir: wks.Dir,
			Attempt:      stage.Attempt{Number: attempt, Maximum: p.maxAttempts},
		})
		p.reporter.StageFinished(types.StageReviewer, attempt, string(rev.Decision), err)
		if err != nil {
			keepRawOutput(err)
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
