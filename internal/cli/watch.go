package cli

import (
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch [run-id]",
	Short: "Watch runs live",
	Long: `Opens the dashboard: every run across every repo, grouped by repo and
			updated live from the daemon. Tab opens the task input, the same one
			plain "dock" starts on. With a run id, will open focused on that
			run — the focused view is being rebuilt on the daemon protocol and is
			not implemented yet.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			return errNotImplemented
		}

		return openApp(cmd.Context(), "", true)
	},
}

func init() {
	rootCmd.AddCommand(watchCmd)
}
