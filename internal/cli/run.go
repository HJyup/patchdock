package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/HJyup/patchdock/internal/config"
	"github.com/HJyup/patchdock/internal/docker"
	"github.com/HJyup/patchdock/internal/pipeline"
	"github.com/HJyup/patchdock/internal/tui"
	"github.com/HJyup/patchdock/internal/types"
	"github.com/spf13/cobra"
)

const AgentTagPrefix = "patchdock-agent"
const logsFile = ".patchdock/logs"

var (
	runIssues []int
	runAll    bool
	runPrompt string
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the pipeline for GitHub issues or a prompt",
	Long: `Runs task(s) through the full pipeline:
		planner → executor → reviewer, each stage in its own
		container, with typed validation at every boundary.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		switch {
		case runPrompt != "":
			return runTask(cmd.Context(), runPrompt)
		case runAll:
			fmt.Println("dock run: (skeleton) would fan out across every open GitHub issue in the repo")
		case len(runIssues) > 0:
			fmt.Printf("dock run: (skeleton) would run the pipeline for issue(s) %v concurrently\n", runIssues)
		default:
			return promptAndRun(cmd.Context())
		}
		return nil
	},
}

func runTask(ctx context.Context, prompt string) error {
	repoAbs, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve current directory: %w", err)
	}

	agentsAbs := filepath.Join(repoAbs, ".patchdock")
	if _, err := os.Stat(agentsAbs); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s is not initialised for patchdock. Run `dock init` first", repoAbs)
		}
		return fmt.Errorf("check %s: %w", agentsAbs, err)
	}

	cfg, err := config.Load(filepath.Join(agentsAbs, "config.yml"))
	if err != nil {
		return fmt.Errorf("%w - edit the file, or regenerate the scaffold with `dock init --force` (overwrites your agent files)", err)
	}

	cli, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("connect to docker: %w. Is the Docker daemon running", err)
	}
	defer cli.Close()

	task, err := types.NewTask(types.Task{Description: prompt})
	if err != nil {
		return fmt.Errorf("invalid task: %w", err)
	}

	agentTag := fmt.Sprintf(`%v%v`, AgentTagPrefix, cfg.ID)
	found, err := cli.ImageExists(ctx, agentTag)
	if err != nil {
		return fmt.Errorf("check image %q: %w. Is the Docker daemon running", agentTag, err)
	}

	progress := tui.New(os.Stdout)
	defer progress.Close()
	progress.Header(task.ID, task.Description)

	if !found {
		if err := buildImage(ctx, cli, agentTag, agentsAbs, progress); err != nil {
			return err
		}
	}

	p := pipeline.NewPipeline(cli, cfg, agentTag, repoAbs, agentsAbs, tui.NewReporter(progress))
	outcome, err := p.Run(ctx, task)
	progress.Close()

	if err != nil {
		return fmt.Errorf("task %s has failed → %w. Check %s", task.ID, err, runReport(outcome))
	}

	if !outcome.Accepted {
		return fmt.Errorf("task %s has failed → reviewer rejected all %d attempt(s). Check %s", task.ID, outcome.Attempts, runReport(outcome))
	}

	progress.Summary(
		fmt.Sprintf("Pipeline finished successfully · %s", plural(outcome.Attempts, "attempt")),
		runReport(outcome))
	return nil
}

func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

// runReport points at the run's own summary, falling back to the logs root when
// the run died before a log directory existed
func runReport(outcome *pipeline.Outcome) string {
	if outcome == nil || outcome.RunDir == "" {
		return logsFile
	}
	return filepath.Join(outcome.RunDir, "run.md")
}

func buildImage(ctx context.Context, cli *docker.Client, imageTag, agentsAbs string, progress *tui.Progress) error {
	if _, err := os.Stat(filepath.Join(agentsAbs, "Dockerfile")); err != nil {
		return fmt.Errorf("image %q not found and %s has no Dockerfile — run `dock init` to scaffold one", imageTag, agentsAbs)
	}

	progress.Start(fmt.Sprintf("Building %s %s", imageTag, progress.Muted("(first run only)")))

	logs, result := cli.Build(ctx, docker.BuildSpec{
		ContextDir: agentsAbs,
		Tag:        imageTag,
		// node_modules is deliberately included: the Dockerfile installs the SDK
		// from it rather than copying SDK source. Run history is not, and grows
		// without bound.
		Exclude: []string{"logs"},
	})

	// Buffered rather than streamed: a successful build has nothing worth
	// reading, and a failed one is unreadable without the whole log
	var buildLog bytes.Buffer
	for line := range logs {
		buildLog.WriteString(line.Text)
	}

	res := <-result
	progress.Finish("", res.Err)

	if res.Err != nil {
		fmt.Fprint(os.Stderr, buildLog.String())
		return fmt.Errorf("failed to build image %q: %w", imageTag, res.Err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().IntSliceVarP(&runIssues, "issue", "i", nil, "GitHub issue number(s) to run, e.g. --i 42,32,12")
	runCmd.Flags().BoolVar(&runAll, "all", false, "run every open GitHub issue in the repository")
	runCmd.Flags().StringVarP(&runPrompt, "prompt", "p", "", "run an ad-hoc prompt instead of a GitHub issue")

	runCmd.MarkFlagsMutuallyExclusive("issue", "all", "prompt")
}
