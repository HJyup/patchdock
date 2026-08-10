package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/HJyup/patchdock/internal/tui"
	"github.com/spf13/cobra"
)

var (
	runPrompt string
	runDetach bool
	runRepo   string
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Queue a run for this repo",
	Long: `Submits a task to the daemon, which takes it through planner →
			executor → reviewer and writes the plan, diff and review of every
			attempt to .patchdock/logs in the target repo.
			Pass the task with --prompt, or omit it to be asked for one.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		prompt := runPrompt

		if prompt == "" {
			if !tui.Interactive(os.Stdin, os.Stdout) {
				return errors.New("no task given, and this is not a terminal to ask on — pass one with --prompt")
			}

			var err error
			prompt, err = tui.Prompt(os.Stdin, os.Stdout)
			if errors.Is(err, tui.ErrPromptCancelled) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("read task: %w", err)
			}
		}

		return submitRun(cmd.Context(), runRepo, prompt, runDetach)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVarP(&runPrompt, "prompt", "p", "", "the task to run")
	runCmd.Flags().BoolVarP(&runDetach, "detach", "d", false, "queue and exit, printing the run id")
	runCmd.Flags().StringVar(&runRepo, "repo", "", "target a repo other than the current directory")
}
