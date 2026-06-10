package cmd

import (
	"fmt"

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
		planner → executor → checks → reviewer, each stage in its own
		container, with live logs streamed to the terminal.`,
	Run: func(cmd *cobra.Command, args []string) {
		switch {
		case runPrompt != "":
			fmt.Printf("patchdock run: (skeleton) would run the pipeline for prompt %q (skipping GitHub issues)\n", runPrompt)
		case runAll:
			fmt.Println("patchdock run: (skeleton) would fan out across every open GitHub issue in the repo")
		case len(runIssues) > 0:
			fmt.Printf("patchdock run: (skeleton) would run the pipeline for issue(s) %v concurrently\n", runIssues)
		default:
			fmt.Println("patchdock run: (skeleton) would open the TUI with the issue picker and a prompt input line")
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().IntSliceVarP(&runIssues, "issue", "i", nil, "GitHub issue number(s) to run, e.g. --i 42,32,12")
	runCmd.Flags().BoolVar(&runAll, "all", false, "run every open GitHub issue in the repository")
	runCmd.Flags().StringVarP(&runPrompt, "prompt", "p", "", "run an ad-hoc prompt instead of a GitHub issue")

	runCmd.MarkFlagsMutuallyExclusive("issue", "all", "prompt")
}
