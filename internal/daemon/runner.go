package daemon

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/HJyup/patchdock/internal/auditlog"
	"github.com/HJyup/patchdock/internal/config"
	"github.com/HJyup/patchdock/internal/credentials"
	"github.com/HJyup/patchdock/internal/daemon/queue"
	"github.com/HJyup/patchdock/internal/docker"
	"github.com/HJyup/patchdock/internal/pipeline"
	"github.com/HJyup/patchdock/internal/stage"
)

const patchdockDir = ".patchdock"

func runPipeline(ctx context.Context, spec queue.RunSpec, rep queue.Reporter) (queue.Outcome, error) {
	dir := filepath.Join(spec.Repo, patchdockDir)

	if _, err := os.Stat(dir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return queue.Outcome{}, fmt.Errorf("%s is not initialised for patchdock, run `dock init` there first", spec.Repo)
		}
		return queue.Outcome{}, fmt.Errorf("check %s: %w", dir, err)
	}

	cfg, err := config.Load(filepath.Join(dir, "config.yml"))
	if err != nil {
		return queue.Outcome{}, err
	}

	cli, err := docker.NewClient()
	if err != nil {
		return queue.Outcome{}, fmt.Errorf("connect to docker: %w. Is the Docker daemon running", err)
	}
	defer cli.Close()

	logger, err := auditlog.New(spec.RunID, dir)
	if err != nil {
		return queue.Outcome{}, fmt.Errorf("open audit log: %w", err)
	}
	defer logger.Close()

	imageTag := cfg.ImageTag()
	found, err := cli.ImageExists(ctx, imageTag)
	if err != nil {
		return queue.Outcome{}, fmt.Errorf("check image %q: %w. Is the Docker daemon running", imageTag, err)
	}
	if !found {
		if err := buildImage(ctx, cli, imageTag, dir, logger); err != nil {
			return queue.Outcome{}, err
		}
	}

	mounts, env, err := credentials.Resolve(cfg.Credentials)
	if err != nil {
		return queue.Outcome{}, fmt.Errorf("load credentials: %w", err)
	}

	stages := stage.NewRunner(cli, stage.RunnerOptions{
		ImageTag:     imageTag,
		PatchdockDir: dir,
		LogWriter:    logger,
		CustomMounts: mounts,
		CustomEnv:    env,
		OnActivity:   rep.StageActivity,
	})

	out, err := pipeline.New(cfg, spec.Repo, stages, logger, rep).Run(ctx, spec.Task)
	if err != nil {
		return queue.Outcome{}, fmt.Errorf("%w. Check %s", err, logger.LogDir)
	}
	if out == nil {
		return queue.Outcome{}, errors.New("pipeline returned no outcome")
	}

	return queue.Outcome{
		Accepted: out.Accepted,
		Branch:   out.Branch,
		Patch:    out.Patch,
	}, nil
}

func buildImage(ctx context.Context, cli *docker.Client, imageTag, dir string, logw io.Writer) error {
	if _, err := os.Stat(filepath.Join(dir, "Dockerfile")); err != nil {
		return fmt.Errorf("image %q not found and %s has no Dockerfile, run `dock init` to scaffold one", imageTag, dir)
	}

	logs, result := cli.Build(ctx, docker.BuildSpec{
		ContextDir: dir,
		ImageTag:   imageTag,
		Exclude:    []string{"logs"},
	})

	for line := range logs {
		if _, err := io.WriteString(logw, line.Text); err != nil {
			return fmt.Errorf("write build log: %w", err)
		}
	}

	if res := <-result; res.Err != nil {
		return fmt.Errorf("build image %q: %w", imageTag, res.Err)
	}

	return nil
}
