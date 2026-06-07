package docker

import (
	"bytes"
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type runResult struct {
	StatusCode int64
	Err        error
}

type runLogs struct {
	Stream string
	Text   string
}

type streamWriter struct {
	ch     chan<- runLogs
	stream string
}

func (w *streamWriter) Write(p []byte) (int, error) {
	for _, line := range bytes.Split(p, []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		w.ch <- runLogs{Stream: w.stream, Text: string(line)}
	}

	return len(p), nil
}

func run(ctx context.Context, cli *client.Client, id string) (<-chan runLogs, <-chan runResult) {
	logs, res := make(chan runLogs), make(chan runResult, 1)

	go func() {
		defer close(logs)
		defer close(res)

		resp, err := cli.ContainerCreate(ctx, &container.Config{Image: id}, nil, nil, nil, "")
		if err != nil {
			res <- runResult{Err: fmt.Errorf("failed to create a container %w", err)}
			return
		}
		defer cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})

		if err = cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
			res <- runResult{Err: fmt.Errorf("failed to start a container %w", err)}
			return
		}

		out, err := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true, ShowStderr: true, Follow: true})
		if err != nil {
			res <- runResult{Err: fmt.Errorf("failed to retrieve logs %w", err)}
			return
		}
		defer out.Close()

		stdoutW := &streamWriter{ch: logs, stream: "stdout"}
		stderrW := &streamWriter{ch: logs, stream: "stderr"}

		if _, err := stdcopy.StdCopy(stdoutW, stderrW, out); err != nil {
			res <- runResult{Err: fmt.Errorf("log stream error: %w", err)}
			return
		}

		statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
		select {
		case err := <-errCh:
			if err != nil {
				res <- runResult{Err: fmt.Errorf("failed container run: %w", err)}
			}
		case status := <-statusCh:
			res <- runResult{StatusCode: status.StatusCode}
		}
	}()

	return logs, res
}
