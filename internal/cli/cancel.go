package cli

import (
	"github.com/spf13/cobra"
)

var cancelCmd = &cobra.Command{
	Use:   "cancel <run-id>",
	Short: "Cancel a queued or running run",
	Long: `Cancels work from any terminal, whichever client submitted it. A
			cancelled run stops at its current stage and its container is removed.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: client.Cancel(args[0]).
		//
		// No auto-start: if there is no daemon there is nothing to cancel, so
		// report that rather than starting one.
		return errNotImplemented
	},
}

func init() {
	rootCmd.AddCommand(cancelCmd)
}
