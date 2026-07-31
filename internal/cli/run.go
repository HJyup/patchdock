package cli

import (
	"github.com/HJyup/patchdock/internal/app"
	"github.com/spf13/cobra"
)

var runPrompt string

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the pipeline for a task",
	Long: `Runs a task through planner → executor → reviewer and writes the
			plan, diff and review of every attempt to .patchdock/logs.
			Pass the task with --prompt, or omit it to be asked for one.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if runPrompt != "" {
			return app.RunTask(cmd.Context(), runPrompt)
		}

		return app.RunPromptInput(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&runPrompt, "prompt", "p", "", "the task to run")
}
