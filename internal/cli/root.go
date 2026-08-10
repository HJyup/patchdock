package cli

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
)

// errNotImplemented marks a command whose wiring exists but whose body is still
// to be written.
var errNotImplemented = errors.New("not implemented yet")

// errNoDaemon is returned by the commands that deliberately do not start one.
var errNoDaemon = errors.New("no daemon running")

// Exit codes, as documented in the README.
const (
	exitFailure  = 1
	exitNoDaemon = 3
)

var (
	rootDetach bool
	rootRepo   string
)

var rootCmd = &cobra.Command{
	Use:          "dock [prompt]",
	SilenceUsage: true,
	Short:        "Run agents against your repo: plan, execute, review — in Docker",
	Long: `Patchdock runs tasks through planner → executor → reviewer, each
			stage in its own container with typed validation between them.
			Without arguments, opens the interactive surface: type a task to
			queue it, queue as many as you like, tab to the live dashboard.
			With an inline prompt, queues it and prints the run id — the
			detached, scriptable form.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		if len(args) == 1 {
			return submitDetached(ctx, rootRepo, args[0])
		}
		if rootDetach {
			return errors.New(`--detach needs an inline prompt: dock -d "…"`)
		}

		return openApp(ctx, rootRepo, false)
	},
}

func init() {
	rootCmd.Flags().BoolVarP(&rootDetach, "detach", "d", false, "queue the inline prompt and print its run id")
	rootCmd.Flags().StringVar(&rootRepo, "repo", "", "target a repo other than the current directory")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if errors.Is(err, errNoDaemon) {
			os.Exit(exitNoDaemon)
		}
		os.Exit(exitFailure)
	}
}
