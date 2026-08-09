package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/HJyup/patchdock/internal/auditlog"
	"github.com/HJyup/patchdock/internal/config"
	"github.com/HJyup/patchdock/internal/credentials"
	"github.com/HJyup/patchdock/internal/docker"
	"github.com/HJyup/patchdock/internal/pipeline"
	"github.com/HJyup/patchdock/internal/stage"
	"github.com/HJyup/patchdock/internal/tui"
	"github.com/HJyup/patchdock/internal/types"
	"github.com/HJyup/patchdock/internal/utils"
)

const patchdockFile = ".patchdock"

var errRejected = errors.New("reviewer rejected every attempt")

// Create a client to the deamon
// get the config
// create a task
// pass all nesserary information to the queue
// deamon should return an http response which works as a SSE connection to the logs

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

	imageTag := cfg.ImageTag()
	found, err := cli.ImageExists(ctx, imageTag)
	if err != nil {
		return fmt.Errorf("check image %q: %w. Is the Docker daemon running", imageTag, err)
	}

	// The logger comes up before the first frame so the header can point at the
	// log directory: a run that dies early is exactly the one whose logs you want
	runID := fmt.Sprintf("%s-%s", task.ID, time.Now().Format("20060102-150405"))
	logger, err := auditlog.New(runID, patchdockDir)
	if err != nil {
		return fmt.Errorf("failed to initialise the logger file: %w", err)
	}
	defer logger.Close()

	logDir := displayPath(repoDir, logger.LogDir)
	started := time.Now()

	progress := tui.New(os.Stdout, cfg.Container.Timeout.Duration())
	defer progress.Close()
	progress.Header(tui.RunInfo{
		Repo:   filepath.Base(repoDir),
		RunID:  task.ID,
		Task:   utils.FirstLine(task.Description),
		LogDir: logDir,
	})

	if !found {
		if err := buildImage(ctx, cli, imageTag, patchdockDir, progress); err != nil {
			return err
		}
	}

	credMounts, credEnv, err := credentials.Resolve(cfg.Credentials)
	if err != nil {
		return fmt.Errorf("load credentials: %w", err)
	}

	reporter := tui.NewReporter(progress)
	runner := stage.NewRunner(cli, stage.RunnerOptions{
		ImageTag:     imageTag,
		PatchdockDir: patchdockDir,
		LogWriter:    logger,
		CustomMounts: credMounts,
		CustomEnv:    credEnv,
		OnActivity:   reporter.StageActivity,
	})

	p := pipeline.New(cfg, repoDir, runner, logger, reporter)
	outcome, err := p.Run(ctx, task)
	progress.Close()

	if err != nil {
		return fmt.Errorf("task %s has failed → %w. Check %s", task.ID, err, logDir)
	}

	progress.Summary(tui.Result{
		Accepted:  outcome.Accepted,
		Attempts:  outcome.Attempts,
		Duration:  time.Since(started),
		Branch:    outcome.Branch,
		Files:     outcome.Patch.Files,
		Additions: outcome.Patch.Additions,
		Deletions: outcome.Patch.Deletions,
		LogDir:    logDir,
	})

	if !outcome.Accepted {
		return errRejected
	}

	return nil
}

func displayPath(base, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return path
	}
	return rel
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
