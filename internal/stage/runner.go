package stage

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/HJyup/patchdock/internal/auth"
	"github.com/HJyup/patchdock/internal/docker"
)

type ContainerRunner interface {
	Run(context.Context, docker.RunSpec) (<-chan docker.LogLine, <-chan docker.RunResult)
}

type Limits struct {
	Timeout   time.Duration
	MaxTokens int
}

type AgentSpec struct {
	AgentFile string
	Limits    Limits
}

// RunnerOptions holds what every stage in one task run shares
type RunnerOptions struct {
	ImageTag     string
	PatchdockDir string
	LogWriter    io.Writer
	Credentials  auth.Credentials
	// kinda call-backish from TypeScript. TBH, I don't know how to handle this in GO
	OnActivity func(activity string)
}

type Attempt struct {
	Number  int
	Maximum int
}

// Runner is task-scoped because its log writer belongs to one audit record
type Runner struct {
	containers ContainerRunner
	options    RunnerOptions
}

func NewRunner(containers ContainerRunner, options RunnerOptions) *Runner {
	return &Runner{containers: containers, options: options}
}

// Used by other stages to ge the output
func decodeOutput[T any](raw []byte, stamp func(*T), build func(T) (T, error)) (T, error) {
	var zero, decoded T
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return zero, ErrOutput{Reason: reasonNotJSON, Err: err, Raw: raw}
	}
	stamp(&decoded)

	out, err := build(decoded)
	if err != nil {
		return zero, ErrOutput{Reason: reasonContract, Err: err, Raw: raw}
	}
	return out, nil
}
