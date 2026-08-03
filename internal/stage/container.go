package stage

import (
	"fmt"
	"strconv"

	"github.com/HJyup/patchdock/internal/docker"
)

type stageMounts struct {
	mounts  []docker.Mount
	claimed map[string]struct{}
}

func NewStageMounts(capacity int) *stageMounts {
	return &stageMounts{
		mounts:  make([]docker.Mount, 0, capacity),
		claimed: make(map[string]struct{}, capacity),
	}
}

func (m *stageMounts) add(mounts ...docker.Mount) error {
	for _, mount := range mounts {
		if _, taken := m.claimed[mount.Target]; taken {
			return fmt.Errorf("mount target %q is already claimed", mount.Target)
		}

		m.claimed[mount.Target] = struct{}{}
		m.mounts = append(m.mounts, mount)
	}

	return nil
}

func (r *Runner) containerSpec(op runOptions, agent AgentSpec) (docker.RunSpec, error) {
	mounts, err := r.containerMounts(op)
	if err != nil {
		return docker.RunSpec{}, err
	}

	env, err := r.containerEnv(op, agent)
	if err != nil {
		return docker.RunSpec{}, err
	}

	return docker.RunSpec{
		Image:   r.options.ImageTag,
		Mounts:  mounts,
		Env:     env,
		Labels:  map[string]string{"patchdock.task-id": op.taskID},
		Timeout: agent.Limits.Timeout,
	}, nil
}

func (r *Runner) containerMounts(op runOptions) ([]docker.Mount, error) {
	credentials := r.options.Credentials.Mounts
	set := NewStageMounts(len(op.mounts) + len(credentials) + 2)

	if err := set.add(
		docker.Mount{Source: op.dir, Target: ioPath, ReadOnly: false},
		docker.Mount{Source: r.options.PatchdockDir, Target: agentsPath, ReadOnly: true},
	); err != nil {
		return nil, err
	}
	if err := set.add(op.mounts...); err != nil {
		return nil, err
	}
	if err := set.add(credentials...); err != nil {
		return nil, err
	}

	return set.mounts, nil
}

func (r *Runner) containerEnv(op runOptions, agent AgentSpec) (map[string]string, error) {
	env := map[string]string{
		"PATCHDOCK_STAGE":   string(op.stage),
		"PATCHDOCK_TASK_ID": op.taskID,
	}
	if agent.AgentFile != "" {
		env["PATCHDOCK_AGENT_FILE"] = agent.AgentFile
	}
	if agent.Limits.MaxTokens > 0 {
		env["PATCHDOCK_TOKEN_BUDGET"] = strconv.Itoa(agent.Limits.MaxTokens)
	}
	if op.attempt > 0 {
		env["PATCHDOCK_ATTEMPT"] = strconv.Itoa(op.attempt)
	}
	if op.maxAttempts > 0 {
		env["PATCHDOCK_MAX_ATTEMPTS"] = strconv.Itoa(op.maxAttempts)
	}

	for key, value := range r.options.Credentials.Env {
		if _, reserved := env[key]; reserved {
			return nil, fmt.Errorf("credential environment variable %q conflicts with stage environment", key)
		}
		env[key] = value
	}

	return env, nil
}
