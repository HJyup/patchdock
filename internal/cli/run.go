package cli

import (
	"github.com/HJyup/patchdock/internal/cli/commands"
	"github.com/spf13/cobra"
)

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
			return commands.RunTask(cmd.Context(), runPrompt)
		default:
			return commands.RunPrompt(cmd.Context())
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&runPrompt, "prompt", "p", "", "run an ad-hoc prompt instead of a GitHub issue")
}
