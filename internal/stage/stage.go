package stage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/HJyup/patchdock/internal/docker"
	"github.com/HJyup/patchdock/internal/types"
)

const (
	input  = "input.json"
	output = "output.json"
)

type opts struct {
	image  string
	stage  types.StageName
	taskID string
	dir    string // Used to create /io mount
	mounts []docker.Mount
}

func runStage(ctx context.Context, c *docker.Client, op opts, inputCnt any) ([]byte, error) {
	if err := os.Mkdir(op.dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create exchange dir: %w", err)
	}

	ioMount := docker.Mount{
		Source:   op.dir,
		Target:   "/io",
		ReadOnly: false,
	}

	mounts := make([]docker.Mount, 0, len(op.mounts)+1)
	for _, mount := range op.mounts {
		if mount.Target == "/io" {
			return nil, errors.New("mount target /io is reserved for the exchange dir")
		}
		mounts = append(mounts, mount)
	}
	mounts = append(mounts, ioMount)

	byteSlice, err := json.Marshal(inputCnt)
	if err != nil {
		return nil, fmt.Errorf("failed to encode input: %w", err)
	}

	inFile := filepath.Join(op.dir, input)
	err = os.WriteFile(inFile, byteSlice, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to write %s: %w", input, err)
	}

	// logs will be then logging to a separate file
	logs, runRes := c.Run(ctx, docker.RunSpec{
		Image:      op.image,
		Mounts:     mounts,
		Env:        map[string]string{"PATCHDOCK_STAGE": string(op.stage)},
		Labels:     map[string]string{"patchdock.task-id": op.taskID},
		Entrypoint: nil,
	})

	// For know, just skip
	for range logs {
	}

	res := <-runRes
	if res.Err != nil {
		return nil, fmt.Errorf("container run failed: %w", res.Err)
	}
	if res.ExitCode != 0 {
		return nil, ErrContainer{ExitCode: res.ExitCode}
	}

	outFile := filepath.Join(op.dir, output)
	content, err := os.ReadFile(outFile)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrOutputMissing{Path: outFile}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", output, err)
	}

	return content, nil
}
