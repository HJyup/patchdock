package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/HJyup/patchdock/internal/auditlog"
	"github.com/HJyup/patchdock/internal/auth"
	"github.com/HJyup/patchdock/internal/config"
	"github.com/HJyup/patchdock/internal/docker"
	"github.com/HJyup/patchdock/internal/pipeline"
	"github.com/HJyup/patchdock/internal/stage"
	"github.com/HJyup/patchdock/internal/tui"
	"github.com/HJyup/patchdock/internal/types"
	"github.com/HJyup/patchdock/internal/utils"
)

const imageTagPrefix = "patchdock-agent"
const patchdockFile = ".patchdock"

func RunTask(ctx context.Context, prompt string) error {
	repoDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve current directory: %w", err)
	}

	patchdockDir := filepath.Join(repoDir, patchdockFile)
	if _, err := os.Stat(patchdockDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s is not initialised for patchdock. Run `dock init` first", patchdockDir)
		}
		return fmt.Errorf("check %s: %w", patchdockDir, err)
	}

	cfgFile := filepath.Join(patchdockDir, "config.yml")
	cfg, err := config.Load(cfgFile)
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

	imageTag := fmt.Sprintf("%s-%s", imageTagPrefix, cfg.Namespace)
	found, err := cli.ImageExists(ctx, imageTag)
	if err != nil {
		return fmt.Errorf("check image %q: %w. Is the Docker daemon running", imageTag, err)
	}

	progress := tui.New(os.Stdout)
	defer progress.Close()
	progress.Header(task.ID, task.Description)

	if !found {
		if err := buildImage(ctx, cli, imageTag, patchdockDir, progress); err != nil {
			return err
		}
	}

	runID := fmt.Sprintf("%s-%s", task.ID, time.Now().Format("20060102-150405"))
	logger, err := auditlog.New(runID, patchdockDir)
	if err != nil {
		return fmt.Errorf("failed to initialise the logger file: %w", err)
	}
	defer logger.Close()

	cred, err := auth.LoadCodex(cfg.Codex)
	if err != nil {
		return fmt.Errorf("load Codex credentials: %w", err)
	}

	reporter := tui.NewReporter(progress)
	runner := stage.NewRunner(cli, stage.RunnerOptions{
		ImageTag:     imageTag,
		PatchdockDir: patchdockDir,
		LogWriter:    logger,
		Credentials:  cred,
		OnActivity:   reporter.StageActivity,
	})

	p := pipeline.New(cfg, repoDir, runner, logger, reporter)
	outcome, err := p.Run(ctx, task)
	progress.Close()

	if err != nil {
		return fmt.Errorf("task %s has failed → %w. Check %s", task.ID, err, logger.LogDir)
	}

	if !outcome.Accepted {
		return fmt.Errorf("task %s has failed → reviewer rejected all %d attempt(s). Check %s", task.ID, outcome.Attempts, logger.LogDir)
	}

	progress.Summary(
		fmt.Sprintf("Pipeline finished successfully · %s", utils.Plural(outcome.Attempts, "attempt")),
		logger.LogDir)
	return nil
}

func buildImage(ctx context.Context, cli *docker.Client, imageTag, patchdockDir string, progress *tui.Progress) error {
	if _, err := os.Stat(filepath.Join(patchdockDir, "Dockerfile")); err != nil {
		return fmt.Errorf("image %q not found and %s has no Dockerfile — run `dock init` to scaffold one", imageTag, patchdockDir)
	}

	progress.Start(fmt.Sprintf("Building %s %s", imageTag, progress.Muted("(first run only)")))
	logs, result := cli.Build(ctx, docker.BuildSpec{
		ContextDir: patchdockDir,
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
