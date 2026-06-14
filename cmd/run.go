package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/HJyup/patchdock/internal/docker"
	"github.com/HJyup/patchdock/internal/stage"
	"github.com/HJyup/patchdock/internal/types"
	"github.com/spf13/cobra"
)

const defaultAgentImage = "patchdock-agent:dev"
const defaultAgentPath = "project-demo"

var (
	runIssues []int
	runAll    bool
	runPrompt string
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the pipeline for GitHub issues or a prompt",
	Long: `Runs task(s) through the full pipeline:
		planner → executor → checks → reviewer, each stage in its own
		container, with live logs streamed to the terminal.`,
	Run: func(cmd *cobra.Command, args []string) {
		switch {
		case runPrompt != "":
			if err := runPlannerProof(cmd.Context(), runPrompt); err != nil {
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

// runPlannerProof temporary function until pipeline doesn't exist
func runPlannerProof(ctx context.Context, prompt string) error {
	repoAbs, err := filepath.Abs(defaultAgentPath)
	if err != nil {
		return fmt.Errorf("resolve repo dir: %w", err)
	}
	agentsAbs := filepath.Join(repoAbs, ".patchdock")
	if _, err := os.Stat(filepath.Join(agentsAbs, "planner.ts")); err != nil {
		return fmt.Errorf("no planner agent at %s (expected planner.ts): %w", agentsAbs, err)
	}

	c, err := docker.NewClient()
	if err != nil {
		return err
	}
	defer c.Close()

	exists, err := c.ImageExists(ctx, defaultAgentImage)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("image %q not found — build it first:\n  docker build -t %s sdk/", defaultAgentImage, defaultAgentPath)
	}

	task, err := types.NewTask(types.Task{Description: prompt})
	if err != nil {
		return fmt.Errorf("invalid task: %w", err)
	}

	exchangeParent, err := os.MkdirTemp("", "patchdock-io-*")
	if err != nil {
		return fmt.Errorf("create exchange parent: %w", err)
	}
	exchangeDir := filepath.Join(exchangeParent, "planner-1")

	plan, err := stage.RunPlanner(ctx, c, stage.PlannerInput{Task: task}, stage.PlannerOpts{
		Image:   defaultAgentImage,
		Dir:     exchangeDir,
		RepoDir: repoAbs,
	})
	if err != nil {
		return fmt.Errorf("planner stage: %w", err)
	}

	out, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}

	fmt.Printf("planner produced a validated Plan\n  exchange dir: %s\n\n%s\n", exchangeDir, out)
	return nil
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().IntSliceVarP(&runIssues, "issue", "i", nil, "GitHub issue number(s) to run, e.g. --i 42,32,12")
	runCmd.Flags().BoolVar(&runAll, "all", false, "run every open GitHub issue in the repository")
	runCmd.Flags().StringVarP(&runPrompt, "prompt", "p", "", "run an ad-hoc prompt instead of a GitHub issue")

	runCmd.MarkFlagsMutuallyExclusive("issue", "all", "prompt")
}
