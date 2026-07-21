package stage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/HJyup/patchdock/internal/auth"
	"github.com/HJyup/patchdock/internal/docker"
	"github.com/HJyup/patchdock/internal/types"
)

const (
	Input           = "input.json"
	Output          = "output.json"
	IOTarget        = "/io"
	AgentsTarget    = "/agents"
	RepoTarget      = "/repo"
	WorkspaceTarget = "/workspace"
)

type runOptions struct {
	stage       types.StageName
	taskID      string
	dir         string
	mounts      []docker.Mount
	attempt     int
	maxAttempts int
}

func (r *Runner) runStage(ctx context.Context, spec StageSpec, op runOptions, inputCnt any) ([]byte, error) {
	if err := os.Mkdir(op.dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create exchange dir: %w", err)
	}

	ioMount := docker.Mount{
		Source:   op.dir,
		Target:   IOTarget,
		ReadOnly: false,
	}

	// Mount the agent definitions explicitly rather than reading them out of
	// the /repo mount: a stage may run without a repo, so agents can't depend
	// on it being present.
	agentMount := docker.Mount{
		Source:   r.options.AgentsDir,
		Target:   AgentsTarget,
		ReadOnly: true,
	}

	mounts := make([]docker.Mount, 0, len(op.mounts)+3)
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
	byteSlice, err := json.MarshalIndent(inputCnt, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to encode input: %w", err)
	}

	inFile := filepath.Join(op.dir, Input)
	err = os.WriteFile(inFile, byteSlice, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to write %s: %w", Input, err)
	}

	env := getEnv(op, spec)
	mounts, err = addCredentials(r.options.Credentials, mounts, env)
	if err != nil {
		return nil, err
	}

	logs, runRes := r.containers.Run(ctx, docker.RunSpec{
		Image:      r.options.Image,
		Mounts:     mounts,
		Env:        env,
		Labels:     map[string]string{"patchdock.task-id": op.taskID},
		Entrypoint: nil,
		Timeout:    spec.Limits.Timeout,
	})

	logWriter := r.options.LogWriter
	if logWriter == nil {
		logWriter = io.Discard
	}
	if err := writeLogEvent(logWriter, map[string]any{
		"source": "patchdock",
		"event":  "stage_started",
		"stage":  op.stage,
	}); err != nil {
		return nil, fmt.Errorf("stage: failed writing to log stream: %w", err)
	}
	for msg := range logs {
		event := make(map[string]any)
		if err := json.Unmarshal([]byte(msg.Text), &event); err != nil {
			event = map[string]any{
				"source":  "container",
				"event":   "message",
				"message": msg.Text,
			}
		}
		event["stage"] = op.stage
		event["stream"] = msg.Stream
		if err := writeLogEvent(logWriter, event); err != nil {
			return nil, fmt.Errorf("stage: failed writing to log stream: %w", err)
		}
	}

	res := <-runRes
	terminalEvent := map[string]any{
		"source":    "patchdock",
		"event":     "stage_finished",
		"stage":     op.stage,
		"exit_code": res.ExitCode,
	}
	if res.Err != nil {
		terminalEvent["level"] = "error"
		terminalEvent["error"] = res.Err.Error()
	} else if res.ExitCode != 0 {
		terminalEvent["level"] = "error"
	} else {
		terminalEvent["level"] = "info"
	}
	if err := writeLogEvent(logWriter, terminalEvent); err != nil {
		return nil, fmt.Errorf("stage: failed writing to log stream: %w", err)
	}
	if res.Err != nil {
		return nil, fmt.Errorf("container run failed: %w", res.Err)
	}
	if res.ExitCode != 0 {
		return nil, ErrContainer{ExitCode: res.ExitCode}
	}

	outFile := filepath.Join(op.dir, Output)
	content, err := os.ReadFile(outFile)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrOutputMissing{Path: outFile}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", Output, err)
	}

	return content, nil
}

func writeLogEvent(w io.Writer, event map[string]any) error {
	if _, exists := event["timestamp"]; !exists {
		event["timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)
	}
	line, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode structured log event: %w", err)
	}
	if _, err := fmt.Fprintln(w, string(line)); err != nil {
		return err
	}
	return nil
}

func getEnv(op runOptions, spec StageSpec) map[string]string {
	env := map[string]string{
		"PATCHDOCK_STAGE":   string(op.stage),
		"PATCHDOCK_TASK_ID": op.taskID,
	}
	if spec.AgentFile != "" {
		env["PATCHDOCK_AGENT_FILE"] = spec.AgentFile
	}
	if spec.Limits.MaxTokens > 0 {
		env["PATCHDOCK_TOKEN_BUDGET"] = strconv.Itoa(spec.Limits.MaxTokens)
	}
	if op.attempt > 0 {
		env["PATCHDOCK_ATTEMPT"] = strconv.Itoa(op.attempt)
	}
	if op.maxAttempts > 0 {
		env["PATCHDOCK_MAX_ATTEMPTS"] = strconv.Itoa(op.maxAttempts)
	}

	return env
}

func addCredentials(credentials auth.Credentials, mounts []docker.Mount, env map[string]string) ([]docker.Mount, error) {
	if len(credentials.Env) == 0 && len(credentials.Mounts) == 0 {
		return mounts, nil
	}
	for key, value := range credentials.Env {
		if _, reserved := env[key]; reserved {
			return nil, fmt.Errorf("credential environment variable %q conflicts with stage environment", key)
		}
		env[key] = value
	}
	existingTargets := make(map[string]struct{}, len(mounts))
	for _, mount := range mounts {
		existingTargets[mount.Target] = struct{}{}
	}
	for _, mount := range credentials.Mounts {
		if _, reserved := existingTargets[mount.Target]; reserved {
			return nil, fmt.Errorf("credential mount target %q conflicts with stage mount", mount.Target)
		}
		existingTargets[mount.Target] = struct{}{}
	}

	return append(mounts, credentials.Mounts...), nil
}
