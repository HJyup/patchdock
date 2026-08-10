package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/HJyup/patchdock/internal/daemon"
	"github.com/HJyup/patchdock/internal/daemon/api"
	"github.com/HJyup/patchdock/internal/runtimedir"
	"github.com/HJyup/patchdock/internal/tui"
)

// submitRun queues prompt against repo — the current directory when empty —
// and, on a terminal, offers to open the dashboard on the freshly queued run.
// Detached or redirected, it prints the bare run id and leaves, which is the
// scriptable form
func submitRun(ctx context.Context, repo, prompt string, detach bool) error {
	if repo == "" {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("resolve current directory: %w", err)
		}
		repo = wd
	}

	// The daemon's working directory is not ours, so a relative path must be
	// resolved on this side of the socket
	repo, err := filepath.Abs(repo)
	if err != nil {
		return fmt.Errorf("resolve repo path %q: %w", repo, err)
	}

	dir, err := runtimedir.Default()
	if err != nil {
		return err
	}

	c, err := daemon.Connect(ctx, &dir)
	if err != nil {
		return err
	}

	resp, err := c.Run(ctx, api.RunRequest{Repo: repo, Prompt: prompt})
	if err != nil {
		return err
	}

	if detach || !tui.Interactive(os.Stdin, os.Stdout) {
		fmt.Println(resp.RunID)
		return nil
	}

	watch, err := tui.ConfirmWatch(os.Stdin, os.Stdout, resp.RunID)
	if err != nil {
		return err
	}
	if !watch {
		fmt.Printf("queued %s — reattach with dock watch\n", resp.RunID)
		return nil
	}

	return tui.Watch(ctx, os.Stdin, os.Stdout, c.StreamRuns)
}
