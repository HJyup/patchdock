package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

type BuildSpec struct {
	ContextDir string
	ImageTag   string
	Exclude    []string
}

type BuildResult struct {
	Err error
}

type Mount struct {
	Source   string // absolute host path
	Target   string // path inside the container, e.g. "/io", "/repo", "/workspace"
	ReadOnly bool
}

// RunSpec describes one container run of a prebuilt image. Run never builds.
type RunSpec struct {
	Image      string
	Mounts     []Mount
	Env        map[string]string // joined to KEY=VALUE by Run
	Labels     map[string]string // e.g. patchdock.run-id
	Entrypoint []string          // nil = image default; set to override
	Timeout    time.Duration     // wall-clock ceiling for the run; 0 = unlimited.
}

type LogLine struct {
	Stream string // "stdout" or "stderr"; empty for build output
	Text   string
}

type RunResult struct {
	ExitCode int64
	Err      error
}

type Client struct {
	cli *client.Client
}

func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to establish connection with docker daemon: %w", err)
	}

	return &Client{
		cli: cli,
	}, nil
}

// Run starts a container from spec and streams its demuxed output.
// Callers must drain the log channel to completion before reading the result
func (c *Client) Run(ctx context.Context, spec RunSpec) (<-chan LogLine, <-chan RunResult) {
	return run(ctx, c.cli, spec)
}

func (c *Client) Build(ctx context.Context, spec BuildSpec) (<-chan LogLine, <-chan BuildResult) {
	return build(ctx, c.cli, spec)
}

func (c *Client) ImageExists(ctx context.Context, imageTag string) (bool, error) {
	list, err := c.cli.ImageList(ctx, image.ListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", imageTag)),
	})
	if err != nil {
		return false, fmt.Errorf("failed to list images for tag %q: %w", imageTag, err)
	}

	return len(list) > 0, nil
}

func (c *Client) Close() error {
	return c.cli.Close()
}
