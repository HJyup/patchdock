package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/HJyup/patchdock/internal/app"
	"github.com/HJyup/patchdock/internal/tui"
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

var rootCmd = &cobra.Command{
	Use:          "dock",
	SilenceUsage: true,
	Short:        "Run agents against your repo: plan, execute, review — in Docker",
	Long: `Patchdock runs a task through planner → executor → reviewer, each
			stage in its own container with typed validation between them.
			Run without arguments to be prompted for a task.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		if !tui.Interactive(os.Stdin, os.Stdout) {
			return errors.New("no task given, and this is not a terminal to ask on — pass one with `dock run -p \"...\"`")
		}

		prompt, err := tui.Prompt(os.Stdin, os.Stdout)
		if errors.Is(err, tui.ErrPromptCancelled) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read task: %w", err)
		}

		return app.RunTask(ctx, prompt)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if errors.Is(err, errNoDaemon) {
			os.Exit(exitNoDaemon)
		}
		os.Exit(exitFailure)
	}
}
