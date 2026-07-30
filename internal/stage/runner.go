package stage

import (
	"context"
	"io"
	"time"

	"github.com/HJyup/patchdock/internal/auth"
	"github.com/HJyup/patchdock/internal/docker"
)

type ContainerRunner interface {
	Run(context.Context, docker.RunSpec) (<-chan docker.LogLine, <-chan docker.Result)
}

type Limits struct {
	Timeout   time.Duration
	MaxTokens int
}

type Spec struct {
	AgentFile string
	Limits    Limits
}

// RunnerOptions holds what every stage in one task run shares
type RunnerOptions struct {
	ImageTag    string
	AgentsDir   string
	LogWriter   io.Writer
	Credentials auth.Credentials
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
