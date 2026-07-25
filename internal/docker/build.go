package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	dockerBuild "github.com/docker/docker/api/types/build"
	"github.com/docker/docker/client"
	"github.com/moby/go-archive"
)

type buildOutput struct {
	// Logs from the output while building an image
	Stream string `json:"stream"`
	Error  string `json:"error"`
	// Getting an image ID after image resolution
	Aux struct {
		ID string `json:"ID"`
	} `json:"aux"`
}

// Build creates a Docker image from the given path and returns two channels:
// logs streams build output lines, result emits a single buildResult with
// the final ImageID or an error. Both channels are closed when the build completes
func build(ctx context.Context, cli *client.Client, spec BuildSpec) (<-chan LogLine, <-chan BuildResult) {
	logs, result := make(chan LogLine), make(chan BuildResult, 1)

	go func() {
		defer close(logs)
		defer close(result)

		tarCxt, err := archive.TarWithOptions(spec.ContextDir, &archive.TarOptions{
			ExcludePatterns: spec.Exclude,
		})

		if err != nil {
			result <- BuildResult{Err: fmt.Errorf("failed to tar a folder: %w", err)}
			return
		}
		defer tarCxt.Close()

		opts := dockerBuild.ImageBuildOptions{ForceRemove: true}
		if spec.Tag != "" {
			opts.Tags = []string{spec.Tag}
		}
		img, err := cli.ImageBuild(ctx, tarCxt, opts)

		if err != nil {
			result <- BuildResult{Err: fmt.Errorf("failed to start image build: %w", err)}
			return
		}
		defer img.Body.Close()

		streamLogs, streamResult := streamBuildLogs(img.Body)
		for msg := range streamLogs {
			logs <- LogLine{
				Stream: "",
				Text:   msg,
			}
		}

		result <- <-streamResult
	}()

	return logs, result
}

func streamBuildLogs(body io.Reader) (<-chan string, <-chan BuildResult) {
	logs, result := make(chan string), make(chan BuildResult, 1)

	go func() {
		var imageID string

		defer close(logs)
		defer close(result)

		decoder := json.NewDecoder(body)

		for {
			var out buildOutput

			err := decoder.Decode(&out)
			if err == io.EOF {
				break
			}

			if err != nil {
				result <- BuildResult{Err: fmt.Errorf("build stream failed: %w", err)}
				return
			}

			if out.Error != "" {
				result <- BuildResult{Err: fmt.Errorf("docker build failed: %s", out.Error)}
				return
			}

			if out.Aux.ID != "" {
				imageID = out.Aux.ID
			}

			if out.Stream != "" {
				logs <- out.Stream
			}
		}

		result <- BuildResult{ImageID: imageID}
	}()

	return logs, result
}
