package docker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/docker/client"
)

type Job struct {
	ID   string
	Path string
}

type phase string

const (
	PhaseBuild phase = "build"
	PhaseRun   phase = "run"
)

type LogLine struct {
	ID     string
	Phase  phase  // "build" or "run" — set by Job.Run as it reads each sub-channel
	Stream string // "stdout"/"stderr" — set by run.go's demuxer; empty for build
	Text   string // the actual line
}

type Result struct {
	ID       string
	ExitCode int64
	Err      error // build/run failure; nil means the container ran to completion
}

func NewJob(id, path string) (*Job, error) {
	if err := checkFolder(path); err != nil {
		return nil, err
	}

	return &Job{ID: id, Path: path}, nil
}

func (j *Job) Run(ctx context.Context, cli *client.Client) (<-chan LogLine, <-chan Result) {
	logs, res := make(chan LogLine), make(chan Result, 1)

	go func() {
		defer close(logs)
		defer close(res)

		buildLogs, buildRes := build(ctx, cli, j.Path)
		for msg := range buildLogs {
			logs <- LogLine{ID: j.ID, Phase: PhaseBuild, Text: msg}
		}

		buildR := <-buildRes
		if buildR.Err != nil {
			res <- Result{ID: j.ID, Err: buildR.Err}
			return
		}

		runLogs, runRes := run(ctx, cli, buildR.ImageID)
		for msg := range runLogs {
			logs <- LogLine{ID: j.ID, Phase: PhaseRun, Stream: msg.Stream, Text: msg.Text}
		}

		runR := <-runRes
		if runR.Err != nil {
			res <- Result{ID: j.ID, Err: runR.Err}
			return
		}

		res <- Result{ID: j.ID, ExitCode: runR.StatusCode}
	}()

	return logs, res
}

// Check folder whether dockerfile exists in the folder
func checkFolder(path string) error {
	dPath := filepath.Join(path, "Dockerfile")
	val, err := os.Stat(dPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("dockerfile doesn't exist: %w", err)
		}
		return fmt.Errorf("failed getting stat for folder: %w", err)
	}
	if val.IsDir() {
		return errors.New("dockerfile is a folder")
	}

	return nil
}
