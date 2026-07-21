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

type StageSpec struct {
	AgentFile string
	Limits    Limits
}

// RunnerOptions contains dependencies shared by every stage in one task run
type RunnerOptions struct {
	Image       string
	AgentsDir   string
	LogWriter   io.Writer
	Credentials auth.Credentials
}

type Attempt struct {
	Number  int
	Maximum int
}

// Runner executes typed stage attempts using shared container machinery.
// A Runner is task-scoped because its log writer belongs to one audit record.
type Runner struct {
	containers ContainerRunner
	options    RunnerOptions
}

func NewRunner(containers ContainerRunner, options RunnerOptions) *Runner {
	return &Runner{containers: containers, options: options}
}
