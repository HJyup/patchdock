package pipeline

import (
	"context"
	"fmt"
	"io"
	"math"
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
	cli        *docker.Client
	cfg        config.Config
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

func NewPipeline(cli *docker.Client, cfg config.Config, image, repoDir, agentsDir string) *Pipeline {
	return &Pipeline{
		cli:        cli,
		cfg:        cfg,
		image:      image,
		repoDir:    repoDir,
		agentsDir:  agentsDir,
		maxRetries: cfg.Retries.Max,
	}
}

func (p *Pipeline) Run(ctx context.Context, task types.Task) (out *Outcome, err error) {
	err = p.validateEnv(ctx)
	if err != nil {
		return nil, err
	}

	env, err := newTaskEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize task environment: %w", err)
	}
	defer env.Cleanup()

	runID := fmt.Sprintf("%s-%s", task.ID, time.Now().Format("20060102-150405"))
	logDir := filepath.Join(p.repoDir, ".patchdock", "logs", runID)

	logger, err := auditlog.NewLogger(logDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize audit logger: %w", err)
	}
	defer logger.Close()

	out = &Outcome{
		TaskID:   task.ID,
		Accepted: false,
	}

	stages, err := p.newStageRunner(logger)
	if err != nil {
		return out, fmt.Errorf("load Codex credentials: %w", err)
	}

	plan, err := stages.RunPlanner(ctx, stage.PlannerRequest{
		Spec:        p.stageSpec(p.cfg.Stages[types.StagePlanner]),
		Input:       stage.PlannerInput{Task: task},
		ExchangeDir: env.PlannerPath(),
		RepoDir:     p.repoDir,
	})
	if err != nil {
		return out, fmt.Errorf("planner stage: %w", err)
	}
	archiveStage(logger, env.PlannerPath())

	out.Plan = plan
	history := newHistory()

	wks, err := workspace.NewWorkspace(p.repoDir, env.WorkspacePath())
	if err != nil {
		return out, fmt.Errorf("failed to create a workspace: %w", err)
	}

	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		res, err := stages.RunExecutor(ctx, stage.ExecutorRequest{
			Spec: p.stageSpec(p.cfg.Stages[types.StageExecutor]),
			Input: stage.ExecutorInput{
				Plan:    plan,
				Reviews: history.Reviews,
			},
			ExchangeDir:  env.ExecutorPath(attempt),
			WorkspaceDir: wks.Dir,
			Attempt:      stage.Attempt{Number: attempt + 1, Maximum: p.maxAttempts()},
		})
		if err != nil {
			return out, fmt.Errorf("executor stage: %w", err)
		}
		archiveStage(logger, env.ExecutorPath(attempt))

		diff, err := wks.Diff(ctx)
		if err != nil {
			return out, fmt.Errorf("executor stage (failed computing diffs): %w", err)
		}
		res.Patch = diff

		history.AddExecution(res)
		out.Execution = res

		rev, err := stages.RunReviewer(ctx, stage.ReviewerRequest{
			Spec: p.stageSpec(p.cfg.Stages[types.StageReviewer]),
			Input: stage.ReviewerInput{
				Plan:             plan,
				ExecutionResults: history.Executions,
				PreviousReviews:  history.Reviews,
			},
			ExchangeDir:  env.ReviewPath(attempt),
			WorkspaceDir: wks.Dir,
			Attempt:      stage.Attempt{Number: attempt + 1, Maximum: p.maxAttempts()},
		})
		if err != nil {
			return out, fmt.Errorf("reviewer stage: %w", err)
		}
		archiveStage(logger, env.ReviewPath(attempt))

		history.AddReview(rev)
		out.Review = rev
		out.Attempts = len(history.Executions)

		if rev.Decision == types.ReviewAccept {
			diffsBytes := []byte(history.Executions[len(history.Executions)-1].Patch)
			if err := logger.WriteDiffs(diffsBytes); err != nil {
				return out, fmt.Errorf("write workspace patch: %w", err)
			}

			out.Accepted = true
			break
		}
	}

	return out, nil
}

func (p *Pipeline) newStageRunner(logWriter io.Writer) (*stage.Runner, error) {
	credentials, err := auth.LoadCodex(p.cfg.Codex)
	if err != nil {
		return nil, err
	}

	return stage.NewRunner(p.cli, stage.RunnerOptions{
		Image:       p.image,
		AgentsDir:   p.agentsDir,
		LogWriter:   logWriter,
		Credentials: credentials,
	}), nil
}

func (p *Pipeline) stageSpec(agentFile string) stage.StageSpec {
	return stage.StageSpec{
		AgentFile: agentFile,
		Limits: stage.Limits{
			Timeout:   p.cfg.Container.Timeout.Duration(),
			MaxTokens: p.cfg.Container.TokenBudget,
		},
	}
}

func (p *Pipeline) maxAttempts() int {
	if p.maxRetries == math.MaxInt {
		return 0
	}
	return p.maxRetries + 1
}

func (p *Pipeline) validateEnv(ctx context.Context) error {
	if p.maxRetries < 0 {
		return fmt.Errorf("maximum amount of retries is incorrect: %v", p.maxRetries)
	}

	if p.maxRetries == 0 {
		p.maxRetries = math.MaxInt
	}

	exists, err := p.cli.ImageExists(ctx, p.image)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("image %q not found — build it first", p.image)
	}

	return nil
}

func archiveStage(logger *auditlog.Logger, dir string) {
	if err := logger.ArchiveStage(dir); err != nil {
		fmt.Fprintf(logger, "audit: failed to archive %s: %v\n", filepath.Base(dir), err)
	}
}
