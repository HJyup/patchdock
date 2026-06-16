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

	return gitDiff(ctx, w.Dir, w.baseCommit)
}

func gitClone(repoDir, dstDir string) error {
	cmd := exec.Command("git", "clone", "--local", repoDir, dstDir)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("stdout/stderr: %s (%v)", errBuf.String(), err)
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
		return "", fmt.Errorf("git diff against %s failed: %v, stderr: %s", baseCommit, err, stderr.String())
	}

	return stdout.String(), nil
}
