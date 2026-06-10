package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show [task-id]",
	Short: "Inspect running and past tasks (stage, tokens, logs)",
	Long: `Opens the inspection TUI over patchdock's run records.
			Without arguments, lists all running and recent tasks — current stage,
			attempt, status, and token usage — and lets you drill into any of them.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			fmt.Printf("patchdock show: (skeleton) would attach to task %s — live stage, token usage, log tail\n", args[0])
			return
		}
		fmt.Println("patchdock show: (skeleton) would open the TUI listing all running/recent tasks (stage, attempt, tokens) for inspection")
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
}
