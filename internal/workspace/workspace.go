package workspace

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const publishTimeout = 60 * time.Second

type Workspace struct {
	Dir        string
	baseCommit string
}

func NewWorkspace(repoDir, dstDir string) (*Workspace, error) {
	if err := gitClone(repoDir, dstDir); err != nil {
		return nil, fmt.Errorf("failed to initialize workspace sandbox: %w", err)
	}

	baseCommit, err := gitRevParse(dstDir, "HEAD")
	if err != nil {
		return nil, fmt.Errorf("failed to lock base commit reference: %w", err)
	}

	return &Workspace{
		Dir:        dstDir,
		baseCommit: baseCommit,
	}, nil
}

func (w *Workspace) Diff(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	if err := gitAddAll(ctx, w.Dir); err != nil {
		return "", fmt.Errorf("failed staging workspace changes: %w", err)
	}

	return gitDiff(ctx, w.Dir, w.baseCommit)
}

func gitClone(repoDir, dstDir string) error {
	cmd := exec.Command("git", "clone", "--local", repoDir, dstDir)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return gitCommandError("git clone", err, errBuf.String())
	}
	return nil
}

func gitRevParse(dir, target string) (string, error) {
	cmd := exec.Command("git", "rev-parse", target)
	cmd.Dir = dir

	outBytes, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(outBytes)), nil
}

func gitDiff(ctx context.Context, dir, baseCommit string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "diff", baseCommit)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", gitCommandError(fmt.Sprintf("git diff against %s", baseCommit), err, stderr.String())
	}

	return stdout.String(), nil
}

func (w *Workspace) Publish(ctx context.Context, branch, message string) error {
	ctx, cancel := context.WithTimeout(ctx, publishTimeout)
	defer cancel()

	if err := gitCheckoutNew(ctx, w.Dir, branch); err != nil {
		return err
	}
	if err := gitAddAll(ctx, w.Dir); err != nil {
		return err
	}
	if err := gitCommit(ctx, w.Dir, message); err != nil {
		return err
	}

	return gitPush(ctx, w.Dir, "origin", branch)
}

func gitCheckoutNew(ctx context.Context, dir, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "checkout", "-b", branch)
	cmd.Dir = dir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return gitCommandError(fmt.Sprintf("git checkout -b %s", branch), err, stderr.String())
	}
	return nil
}

func gitCommit(ctx context.Context, dir, message string) error {
	cmd := exec.CommandContext(ctx, "git", "commit", "-m", message)
	cmd.Dir = dir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return gitCommandError("git commit", err, stderr.String())
	}
	return nil
}

func gitPush(ctx context.Context, dir, remote, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "push", remote, branch)
	cmd.Dir = dir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return gitCommandError(fmt.Sprintf("git push %s %s", remote, branch), err, stderr.String())
	}
	return nil
}

func gitAddAll(ctx context.Context, dir string) error {
	cmd := exec.CommandContext(ctx, "git", "add", "-A")
	cmd.Dir = dir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return gitCommandError("git add", err, stderr.String())
	}
	return nil
}

func gitCommandError(operation string, err error, stderr string) error {
	stderr = strings.TrimSpace(stderr)
	if stderr == "" {
		return fmt.Errorf("%s: %w", operation, err)
	}

	return fmt.Errorf("%s: %w: %s", operation, err, stderr)
}
