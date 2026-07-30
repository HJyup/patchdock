package cli

import (
	"os"

	"github.com/HJyup/patchdock/internal/cli/commands"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:          "dock",
	SilenceUsage: true,
	Short:        "A typed agent-pipeline runtime: plan, execute, review — in Docker",
	Long: `Patchdock drives a fixed pipeline against a code repository.
		Run without arguments to open the TUI: pick GitHub issues or enter a
		prompt, watch concurrent tasks move through the pipeline, inspect
		plans and diffs, and gate pull requests.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return commands.RunPrompt(cmd.Context())
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
