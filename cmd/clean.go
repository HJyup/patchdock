package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var cleanDryRun bool

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove unused patchdock containers, stale workspaces, and old logs",
	Long: `Cleans up everything patchdock leaves behind:
		  - containers labeled patchdock.task-id that no live run owns.
		  - stale workspace copies under .patchdock/work/
		  - audit logs older than the retention period in config.yml.`,
	Run: func(cmd *cobra.Command, args []string) {
		if cleanDryRun {
			fmt.Println("patchdock clean: (skeleton) would list unused containers, stale workspaces, and expired logs — without deleting (--dry-run)")
			return
		}
		fmt.Println("patchdock clean: (skeleton) would remove unused containers, stale workspaces, and logs older than the configured retention")
	},
}

func init() {
	rootCmd.AddCommand(cleanCmd)

	cleanCmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "show what would be removed without deleting anything")
}
