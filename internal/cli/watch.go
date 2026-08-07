package cli

import (
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch [run-id]",
	Short: "Watch runs live",
	Long: `Without an argument, shows every run across every repo — the same
			read-only view as plain "dock". With a run id, opens focused on that
			run, which is how you get back after detaching from "dock run".`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: connect, GET the current runs to paint the initial view, then
		// subscribe to the event stream from the last sequence seen.
		//
		// With a run id, filter the subscription to that run and exit with the
		// run's outcome code once it reaches a terminal state, so detaching and
		// reattaching is equivalent to having stayed attached.
		return errNotImplemented
	},
}

func init() {
	rootCmd.AddCommand(watchCmd)
}
