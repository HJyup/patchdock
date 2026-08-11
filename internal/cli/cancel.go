package cli

import (
	"errors"
	"fmt"

	"github.com/HJyup/patchdock/internal/daemon/client"
	"github.com/HJyup/patchdock/internal/runtimedir"
	"github.com/spf13/cobra"
)

var cancelCmd = &cobra.Command{
	Use:   "cancel <run-id>",
	Short: "Cancel a queued or running run",
	Long: `Cancels work from any terminal, whichever client submitted it. A
			cancelled run stops at its current stage and its container is removed.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := runtimedir.Default()
		if err != nil {
			return err
		}

		runID := args[0]
		if err := client.New(dir.Socket()).Cancel(cmd.Context(), runID); err != nil {
			if errors.Is(err, client.ErrNoDaemon) || errors.Is(err, client.ErrNotListening) {
				return errNoDaemon
			}
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "cancelled %s\n", runID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(cancelCmd)
}
