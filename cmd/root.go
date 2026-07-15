package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "patchdock",
	Short: "A typed agent-pipeline runtime: plan, execute, review — in Docker",
	Long: `Patchdock drives a fixed pipeline against a code repository.
		Run without arguments to open the TUI: pick GitHub issues or enter a
		prompt, watch concurrent tasks move through the pipeline, inspect
		plans and diffs, and gate pull requests.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("patchdock: (skeleton) would open the TUI — issue picker, prompt input, and the live view of running tasks")
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
