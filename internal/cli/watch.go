package cli

import (
	"os"

	"github.com/HJyup/patchdock/internal/daemon"
	"github.com/HJyup/patchdock/internal/runtimedir"
	"github.com/HJyup/patchdock/internal/tui"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch [run-id]",
	Short: "Watch runs live",
	Long: `Opens the dashboard: every run across every repo, grouped by repo and
			updated live from the daemon. With a run id, will open focused on that
			run — the focused view is being rebuilt on the daemon protocol and is
			not implemented yet.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			return errNotImplemented
		}

		dir, err := runtimedir.Default()
		if err != nil {
			return err
		}

		c, err := daemon.Connect(cmd.Context(), &dir)
		if err != nil {
			return err
		}

		return tui.Watch(cmd.Context(), os.Stdin, os.Stdout, c.StreamRuns)
	},
}

func init() {
	rootCmd.AddCommand(watchCmd)
}
