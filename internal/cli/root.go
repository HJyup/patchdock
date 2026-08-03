package cli

import (
	"os"

	"github.com/HJyup/patchdock/internal/app"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:          "dock",
	SilenceUsage: true,
	Short:        "Run agents against your repo: plan, execute, review — in Docker",
	Long: `Patchdock runs a task through planner → executor → reviewer, each
			stage in its own container with typed validation between them.
			Run without arguments to be prompted for a task.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return app.RunPromptInput(cmd.Context())
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
