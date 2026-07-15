package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/HJyup/patchdock/internal/config"
	"github.com/HJyup/patchdock/internal/docker"
	"github.com/HJyup/patchdock/internal/pipeline"
	"github.com/HJyup/patchdock/internal/types"
	"github.com/spf13/cobra"
)

const AgentName = "patchdock-agent:dev"
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
			fmt.Println("patchdock run: (skeleton) would fan out across every open GitHub issue in the repo")
		case len(runIssues) > 0:
			fmt.Printf("patchdock run: (skeleton) would run the pipeline for issue(s) %v concurrently\n", runIssues)
		default:
			fmt.Println("patchdock run: (skeleton) would open the TUI with the issue picker and a prompt input line")
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
			return fmt.Errorf("%s is not initialised for patchdock. Run `patchdock init` first", repoAbs)
		}
		return fmt.Errorf("check %s: %w", agentsAbs, err)
	}

	cfg, err := config.Load(filepath.Join(agentsAbs, "config.yml"))
	if err != nil {
		return fmt.Errorf("%w - edit the file, or regenerate the scaffold with `patchdock init --force` (overwrites your agent files)", err)
	}

	cli, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("connect to docker: %w. Is the Docker daemon running?", err)
	}
	defer cli.Close()

	task, err := types.NewTask(types.Task{Description: prompt})
	if err != nil {
		return fmt.Errorf("invalid task: %w", err)
	}

	found, err := cli.ImageExists(ctx, AgentName)
	if err != nil {
		return fmt.Errorf("check image %q: %w. Is the Docker daemon running?", AgentName, err)
	}

	if !found {
		if err := buildImage(ctx, cli, AgentName, repoAbs); err != nil {
			return err
		}
	}

	p := pipeline.NewPipeline(cli, cfg, AgentName, repoAbs, agentsAbs)
	outcome, err := p.Run(ctx, task)
	if err != nil {
		return fmt.Errorf("task %s has failed → %w. Check %s", task.ID, err, logsFile)
	}

	if !outcome.Accepted {
		return fmt.Errorf("task %s has failed → reviewer rejected all %d attempt(s). Check %s", task.ID, outcome.Attempts, logsHint)
	}

	fmt.Printf("Task %s has finished successfully (attempts: %d)\n", task.ID, outcome.Attempts)
	return nil
}

func buildImage(ctx context.Context, cli *docker.Client, image, repoDir string) error {
	sdkDir := filepath.Join(repoDir, "sdk")
	if _, err := os.Stat(filepath.Join(sdkDir, "Dockerfile")); err != nil {
		return fmt.Errorf("image %q not found and this repo has no recipe for it — build it from a patchdock checkout:\n  docker build -t %s <patchdock>/sdk", image, image)
	}

	fmt.Printf("image %q not found — building from %s (first run only)\n", image, sdkDir)

	logs, result := cli.Build(ctx, docker.BuildSpec{
		ContextDir: sdkDir,
		Tag:        image,
		Exclude:    []string{"node_modules"},
	})
	for line := range logs {
		fmt.Print(line.Text)
	}

	if res := <-result; res.Err != nil {
		return fmt.Errorf("failed to build image %q: %w", image, res.Err)
	}

	fmt.Printf("image %q ready\n\n", image)
	return nil
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().IntSliceVarP(&runIssues, "issue", "i", nil, "GitHub issue number(s) to run, e.g. --i 42,32,12")
	runCmd.Flags().BoolVar(&runAll, "all", false, "run every open GitHub issue in the repository")
	runCmd.Flags().StringVarP(&runPrompt, "prompt", "p", "", "run an ad-hoc prompt instead of a GitHub issue")

	runCmd.MarkFlagsMutuallyExclusive("issue", "all", "prompt")
}
