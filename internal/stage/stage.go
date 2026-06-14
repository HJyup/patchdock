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
	input           = "input.json"
	output          = "output.json"
	IOTarget        = "/io"
	AgentsTarget    = "/agents"
	RepoTarget      = "/repo"
	WorkspaceTarget = "/workspace"
)

type opts struct {
	image      string
	stage      types.StageName
	taskID     string
	dir        string
	mounts     []docker.Mount
	agentsPath string
}

func runStage(ctx context.Context, c *docker.Client, op opts, inputCnt any) ([]byte, error) {
	if err := os.Mkdir(op.dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create exchange dir: %w", err)
	}

	ioMount := docker.Mount{
		Source:   op.dir,
		Target:   IOTarget,
		ReadOnly: false,
	}

	// Mounts agents ts files from the .patchdock.
	// Yeah, we have .patchdock already from ./repo mount but what if we wanna define a new stage where
	// repo is not used? So it's better not to try to parse them from there.
	agentMount := docker.Mount{
		Source:   op.agentsPath,
		Target:   AgentsTarget,
		ReadOnly: true,
	}

	mounts := make([]docker.Mount, 0, len(op.mounts)+2)
	for _, mount := range op.mounts {
		if mount.Target == IOTarget {
			return nil, fmt.Errorf("mount target %v is reserved for the exchange dir", IOTarget)
		}
		if mount.Target == AgentsTarget {
			return nil, fmt.Errorf("mount target %v is reserved for the agents definitions", AgentsTarget)
		}
		mounts = append(mounts, mount)
	}
	mounts = append(mounts, ioMount)
	mounts = append(mounts, agentMount)

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
		Image:  op.image,
		Mounts: mounts,
		Env: map[string]string{
			"PATCHDOCK_STAGE":   string(op.stage),
			"PATCHDOCK_TASK_ID": op.taskID,
		},
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
