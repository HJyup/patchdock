package cmd

import (
	"context"
	"encoding/json"
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
	Run: func(cmd *cobra.Command, args []string) {
		switch {
		case runPrompt != "":
			if err := runTask(cmd.Context(), runPrompt); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
		case runAll:
			fmt.Println("patchdock run: (skeleton) would fan out across every open GitHub issue in the repo")
		case len(runIssues) > 0:
			fmt.Printf("patchdock run: (skeleton) would run the pipeline for issue(s) %v concurrently\n", runIssues)
		default:
			fmt.Println("patchdock run: (skeleton) would open the TUI with the issue picker and a prompt input line")
		}
	},
}

func runTask(ctx context.Context, prompt string) error {
	repoAbs, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get absolute path")
	}

	agentsAbs := filepath.Join(repoAbs, ".patchdock")
	if _, err := os.Stat(agentsAbs); err != nil {
		return fmt.Errorf("agents dir not found at %s: %w", agentsAbs, err)
	}

	cfg, err := config.Load(filepath.Join(agentsAbs, "config.yml"))
	if err != nil {
		return err
	}

	cli, err := docker.NewClient()
	if err != nil {
		return err
	}
	defer cli.Close()

	task, err := types.NewTask(types.Task{Description: prompt})
	if err != nil {
		return fmt.Errorf("invalid task: %w", err)
	}

	found, err := cli.ImageExists(ctx, AgentName)
	if err != nil {
		return fmt.Errorf("failed to check whether image %q exists: %w", AgentName, err)
	}

	if !found {
		if err := buildImage(ctx, cli, AgentName, repoAbs); err != nil {
			return err
		}
	}

	p := pipeline.NewPipeline(cli, cfg, AgentName, repoAbs, agentsAbs)
	outcome, err := p.Run(ctx, task)
	if err != nil {
		return fmt.Errorf("pipeline: %w", err)
	}

	out, err := json.Marshal(outcome)
	if err != nil {
		return err
	}

	fmt.Printf("pipeline finished — accepted=%v, attempts=%d\n\n%s\n", outcome.Accepted, outcome.Attempts, out)
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
