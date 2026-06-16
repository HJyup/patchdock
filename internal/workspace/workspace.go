package workspace

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type Workspace struct {
	Dir        string
	baseCommit string
}

func NewWorkspace(repoDir, dstDir string) (*Workspace, error) {
	cloneCmd := exec.Command("git", "clone", "--local", repoDir, dstDir)
	var errBuf bytes.Buffer
	cloneCmd.Stderr = &errBuf

	if err := cloneCmd.Run(); err != nil {
		return nil, fmt.Errorf("Clone failed: %v\nError: %s\n", err, errBuf.String())
	}

	revCmd := exec.Command("git", "rev-parse", "HEAD")
	revCmd.Dir = dstDir

	baseCommitBytes, err := revCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("Failed to get HEAD commit: %v\n", err)
	}

	baseCommit := strings.TrimSpace(string(baseCommitBytes))
	return &Workspace{
		Dir:        dstDir,
		baseCommit: baseCommit,
	}, nil
}

func (w *Workspace) Diff() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	addCmd := exec.CommandContext(ctx, "git", "add", "-A")
	addCmd.Dir = w.Dir
	if err := addCmd.Run(); err != nil {
		return "", fmt.Errorf("failed to stage files (git add -A): %w", err)
	}

	diffCmd := exec.CommandContext(ctx, "git", "diff", w.baseCommit)
	diffCmd.Dir = w.Dir

	var stdout, stderr bytes.Buffer
	diffCmd.Stdout = &stdout
	diffCmd.Stderr = &stderr

	if err := diffCmd.Run(); err != nil {
		return "", fmt.Errorf("git diff failed: %v, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}
