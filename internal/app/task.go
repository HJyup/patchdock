package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/HJyup/patchdock/internal/config"
	"github.com/HJyup/patchdock/internal/docker"
	"github.com/HJyup/patchdock/internal/pipeline"
	"github.com/HJyup/patchdock/internal/tui"
	"github.com/HJyup/patchdock/internal/types"
)

const imageTagPrefix = "patchdock-agent"
const logsFile = ".patchdock/logs"

func RunTask(ctx context.Context, prompt string) error {
	repoDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve current directory: %w", err)
	}

	patchdockDir := filepath.Join(repoDir, ".patchdock")
	if _, err := os.Stat(patchdockDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s is not initialised for patchdock. Run `dock init` first", patchdockDir)
		}
		return fmt.Errorf("check %s: %w", patchdockDir, err)
	}

	patchdockCfg, err := config.Load(filepath.Join(patchdockDir, "config.yml"))
	if err != nil {
		return fmt.Errorf("%w - edit the file, or regenerate the scaffold with `dock init --force` (overwrites your agent files)", err)
	}

	cli, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("connect to docker: %w. Is the Docker daemon running", err)
	}
	defer cli.Close()

	task, err := types.NewTask(types.Task{Description: prompt})
	if err != nil {
		return fmt.Errorf("invalid task: %w", err)
	}

	imageTag := fmt.Sprintf(`%v%v`, imageTagPrefix, patchdockCfg.ID)
	found, err := cli.ImageExists(ctx, imageTag)
	if err != nil {
		return fmt.Errorf("check image %q: %w. Is the Docker daemon running", imageTag, err)
	}

	progress := tui.New(os.Stdout)
	defer progress.Close()
	progress.Header(task.ID, task.Description)

	if !found {
		if err := buildImage(ctx, cli, imageTag, repoDir, progress); err != nil {
			return err
		}
	}

	p := pipeline.NewPipeline(cli, patchdockCfg, imageTag, repoDir, patchdockDir, tui.NewReporter(progress))
	outcome, err := p.Run(ctx, task)
	progress.Close()

	if err != nil {
		return fmt.Errorf("task %s has failed → %w. Check %s", task.ID, err, runReport(outcome))
	}

	if !outcome.Accepted {
		return fmt.Errorf("task %s has failed → reviewer rejected all %d attempt(s). Check %s", task.ID, outcome.Attempts, runReport(outcome))
	}

	progress.Summary(
		fmt.Sprintf("Pipeline finished successfully · %s", plural(outcome.Attempts, "attempt")),
		runReport(outcome))
	return nil
}

func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

// runReport points at the run's own summary, falling back to the logs root when
// the run died before a log directory existed
func runReport(outcome *pipeline.Outcome) string {
	if !outcome.Accepted {
		return logsFile
	}
	return filepath.Join(outcome.RunDir, "run.md")
}

func buildImage(ctx context.Context, cli *docker.Client, imageTag, agentsAbs string, progress *tui.Progress) error {
	if _, err := os.Stat(filepath.Join(agentsAbs, "Dockerfile")); err != nil {
		return fmt.Errorf("image %q not found and %s has no Dockerfile — run `dock init` to scaffold one", imageTag, agentsAbs)
	}

	progress.Start(fmt.Sprintf("Building %s %s", imageTag, progress.Muted("(first run only)")))
	logs, result := cli.Build(ctx, docker.BuildSpec{
		ContextDir: agentsAbs,
		ImageTag:   imageTag,
		Exclude:    []string{"logs"},
	})

	var buildLog bytes.Buffer
	for line := range logs {
		buildLog.WriteString(line.Text)
	}

	res := <-result
	progress.Finish("", res.Err)

	if res.Err != nil {
		fmt.Fprint(os.Stderr, buildLog.String())
		return fmt.Errorf("failed to build image %q: %w", imageTag, res.Err)
	}

	return nil
}
