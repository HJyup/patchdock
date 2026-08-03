package stage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/HJyup/patchdock/internal/docker"
	"github.com/HJyup/patchdock/internal/stage/events"
	"github.com/HJyup/patchdock/internal/types"
)

const (
	inputFile  = "input.json"
	outputFile = "output.json"
)

const (
	ioPath        = "/io"
	agentsPath    = "/agents"
	repoPath      = "/repo"
	workspacePath = "/workspace"
)

type runOptions struct {
	stage       types.StageName
	taskID      string
	dir         string
	mounts      []docker.Mount
	attempt     int
	maxAttempts int
}

func (r *Runner) runStage(ctx context.Context, agent AgentSpec, op runOptions, inputCnt any) ([]byte, error) {
	if err := os.Mkdir(op.dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create exchange dir: %w", err)
	}

	byteSlice, err := json.MarshalIndent(inputCnt, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to encode input: %w", err)
	}

	inFile := filepath.Join(op.dir, inputFile)
	err = os.WriteFile(inFile, byteSlice, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to write %s: %w", inputFile, err)
	}

	containerSpec, err := r.containerSpec(op, agent)
	if err != nil {
		return nil, err
	}

	logs, runRes := r.containers.Run(ctx, containerSpec)

	stream := events.New(r.options.LogWriter, op.stage, r.options.OnActivity)
	if err := stream.Started(); err != nil {
		return nil, err
	}
	for msg := range logs {
		if err := stream.Line(msg); err != nil {
			return nil, err
		}
	}

	res := <-runRes
	if err := stream.Finished(res); err != nil {
		return nil, err
	}

	if res.Err != nil {
		return nil, fmt.Errorf("container run failed: %w", res.Err)
	}
	if res.ExitCode != 0 {
		return nil, ErrContainer{ExitCode: res.ExitCode}
	}

	outFile := filepath.Join(op.dir, outputFile)
	content, err := os.ReadFile(outFile)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrOutputMissing{Path: outFile}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", outputFile, err)
	}

	return content, nil
}
