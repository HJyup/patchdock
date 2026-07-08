package docker

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type streamWriter struct {
	ctx    context.Context
	ch     chan<- LogLine
	stream string
}

func (w *streamWriter) Write(p []byte) (int, error) {
	for _, line := range bytes.Split(p, []byte("\n")) {
		if len(line) == 0 {
			continue
		}

		select {
		case w.ch <- LogLine{Stream: w.stream, Text: string(line)}:
		case <-w.ctx.Done():
			return 0, w.ctx.Err()
		}
	}

	return len(p), nil
}

func envSlice(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}
	out := make([]string, 0, len(env))
	for k, v := range env {
		out = append(out, k+"="+v)
	}

	return out
}

func run(ctx context.Context, cli *client.Client, spec RunSpec) (<-chan LogLine, <-chan Result) {
	logs, res := make(chan LogLine), make(chan Result, 1)

	go func() {
		defer close(logs)
		defer close(res)

		if spec.Timeout > 0 {
              var cancel context.CancelFunc
              ctx, cancel = context.WithTimeout(ctx, spec.Timeout)
              defer cancel()
      }

		for _, m := range spec.Mounts {
			if !filepath.IsAbs(m.Source) {
				res <- Result{Err: fmt.Errorf("mount source %q must be an absolute path", m.Source)}
				return
			}
			if _, err := os.Stat(m.Source); err != nil {
				res <- Result{Err: fmt.Errorf("mount source %q: %w", m.Source, err)}
				return
			}
		}

		mounts := make([]mount.Mount, 0, len(spec.Mounts))
		for _, m := range spec.Mounts {
			mounts = append(mounts, mount.Mount{
				Type:     mount.TypeBind,
				Source:   m.Source,
				Target:   m.Target,
				ReadOnly: m.ReadOnly,
			})
		}

		resp, err := cli.ContainerCreate(ctx,
			&container.Config{
				Image:      spec.Image,
				Env:        envSlice(spec.Env),
				Labels:     spec.Labels,
				Entrypoint: spec.Entrypoint,
			},
			&container.HostConfig{Mounts: mounts},
			nil, nil, "")
		if err != nil {
			res <- Result{Err: fmt.Errorf("failed to create a container: %w", err)}
			return
		}

		cleanupCtx := context.WithoutCancel(ctx)
		defer cli.ContainerRemove(cleanupCtx, resp.ID, container.RemoveOptions{Force: true})

		if err = cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
			res <- Result{Err: fmt.Errorf("failed to start a container: %w", err)}
			return
		}

		out, err := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true, ShowStderr: true, Follow: true})
		if err != nil {
			res <- Result{Err: fmt.Errorf("failed to retrieve logs: %w", err)}
			return
		}
		defer out.Close()

		stdoutW := &streamWriter{ctx: ctx, ch: logs, stream: "stdout"}
		stderrW := &streamWriter{ctx: ctx, ch: logs, stream: "stderr"}

		if _, err := stdcopy.StdCopy(stdoutW, stderrW, out); err != nil {
			if ctx.Err() != nil {
				res <- Result{Err: ctx.Err()}
				return
			}
			res <- Result{Err: fmt.Errorf("log stream error: %w", err)}
			return
		}

		statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
		select {
		case status := <-statusCh:
			res <- Result{ExitCode: status.StatusCode}
		case err := <-errCh:
			res <- Result{Err: fmt.Errorf("failed container run: %w", err)}
		case <-ctx.Done():
			res <- Result{Err: ctx.Err()}
		}
	}()

	return logs, res
}
