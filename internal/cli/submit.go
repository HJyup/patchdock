package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/HJyup/patchdock/internal/daemon"
	"github.com/HJyup/patchdock/internal/daemon/api"
	"github.com/HJyup/patchdock/internal/daemon/client"
	"github.com/HJyup/patchdock/internal/runtimedir"
	"github.com/HJyup/patchdock/internal/tui"
)

// resolveRepo turns repo — or the current directory when it is empty — into
// an absolute path. The daemon's working directory is not ours, so a relative
// path must be resolved on this side of the socket
func resolveRepo(repo string) (string, error) {
	if repo == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("resolve current directory: %w", err)
		}
		return wd, nil
	}

	abs, err := filepath.Abs(repo)
	if err != nil {
		return "", fmt.Errorf("resolve repo path %q: %w", repo, err)
	}
	return abs, nil
}

func connect(ctx context.Context) (*client.Client, error) {
	dir, err := runtimedir.Default()
	if err != nil {
		return nil, err
	}

	return daemon.Connect(ctx, &dir)
}

// submitDetached queues one prompt and prints the bare run id — the form
// scripts can capture
func submitDetached(ctx context.Context, repo, prompt string) error {
	repo, err := resolveRepo(repo)
	if err != nil {
		return err
	}

	c, err := connect(ctx)
	if err != nil {
		return err
	}

	resp, err := c.Run(ctx, api.RunRequest{Repo: repo, Prompt: prompt})
	if err != nil {
		return err
	}

	fmt.Println(resp.RunID)
	return nil
}

// openApp starts the interactive surface, either on the task input or on the
// dashboard. Submissions from the input land in repo
func openApp(ctx context.Context, repo string, startOnWatch bool) error {
	repo, err := resolveRepo(repo)
	if err != nil {
		return err
	}

	c, err := connect(ctx)
	if err != nil {
		return err
	}

	return tui.App(ctx, os.Stdin, os.Stdout, tui.AppOptions{
		Repo: repo,
		Submit: func(ctx context.Context, prompt string) (string, error) {
			resp, err := c.Run(ctx, api.RunRequest{Repo: repo, Prompt: prompt})
			return resp.RunID, err
		},
		Stream:       c.StreamRuns,
		StartOnWatch: startOnWatch,
	})
}
