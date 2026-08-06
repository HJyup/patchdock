package cli

import (
	"github.com/HJyup/patchdock/internal/app"
	"github.com/spf13/cobra"
)

var (
	runPrompt string
	runDetach bool
	runRepo   string
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Queue a run for this repo",
	Long: `Queues a task through planner → executor → reviewer and writes the
			plan, diff and review of every attempt to .patchdock/logs.
			Pass the task with --prompt, or omit it to be asked for one.
			Ctrl-C detaches; the run keeps going.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: both paths become client.Submit against the daemon.
		//   --repo, or the cwd, resolved to an ABSOLUTE path — the daemon's
		//   working directory is wherever it was spawned.
		//   with -d:  print the run id on stdout, hints on stderr, exit 0.
		//   without:  subscribe to this run's events, render, and exit with the
		//             run's outcome code.
		if runDetach || runRepo != "" {
			return errNotImplemented
		}

		if runPrompt != "" {
			return app.RunTask(cmd.Context(), runPrompt)
		}

		return app.RunPromptInput(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVarP(&runPrompt, "prompt", "p", "", "the task to run")
	runCmd.Flags().BoolVarP(&runDetach, "detach", "d", false, "queue and exit, printing the run id")
	runCmd.Flags().StringVar(&runRepo, "repo", "", "target a repo other than the current directory")
}
