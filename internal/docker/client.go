package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

// BuildSpec describes one image build: tar ContextDir, send it to the
// daemon, name the result Tag.
type BuildSpec struct {
	ContextDir string
	Dockerfile string
	Tag        string
}

// BuildResult is the terminal outcome of a Build.
type BuildResult struct {
	ImageID string // the daemon's content-addressed ID
	Err     error
}

// Mount shares one host directory into the container.
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
	Labels     map[string]string // e.g. patchdock.task-id
	Entrypoint []string          // nil = image default; set to override (check mode)
}

// LogLine is one demuxed output line from a build or run.
type LogLine struct {
	Stream string // "stdout" or "stderr"; empty for build output
	Text   string
}

// Result is the terminal outcome of a Run.
type Result struct {
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
func (c *Client) Run(ctx context.Context, spec RunSpec) (<-chan LogLine, <-chan Result) {
	return run(ctx, c.cli, spec)
}

// Build tars spec.ContextDir, builds it, and tags the result.
func (c *Client) Build(ctx context.Context, spec BuildSpec) (<-chan LogLine, <-chan BuildResult) {
	return build(ctx, c.cli, spec)
}

// ImageExists reports whether an image with the given tag is present on the daemon.
func (c *Client) ImageExists(ctx context.Context, tag string) (bool, error) {
	list, err := c.cli.ImageList(ctx, image.ListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", tag)),
	})
	if err != nil {
		return false, fmt.Errorf("failed to list images for tag %q: %w", tag, err)
	}

	return len(list) > 0, nil
}

func (c *Client) Close() error {
	return c.cli.Close()
}
