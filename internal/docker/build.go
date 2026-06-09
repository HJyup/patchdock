package docker

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	dockerBuild "github.com/docker/docker/api/types/build"
	"github.com/docker/docker/client"
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

type buildResult struct {
	ImageID string
	Err     error
}

// Build creates a Docker image from the given path and returns two channels:
// logs streams build output lines, result emits a single buildResult with
// the final ImageID or an error. Both channels are closed when the build completes
func build(ctx context.Context, cli *client.Client, path string) (<-chan string, <-chan buildResult) {
	logs, result := make(chan string), make(chan buildResult, 1)

	go func() {
		defer close(logs)
		defer close(result)

		tarCxt, err := tarDir(path)
		if err != nil {
			result <- buildResult{Err: fmt.Errorf("failed to tar a folder: %w", err)}
			return
		}
		defer tarCxt.Close()

		img, err := cli.ImageBuild(ctx, tarCxt, dockerBuild.ImageBuildOptions{
			ForceRemove: true,
		})
		if err != nil {
			result <- buildResult{Err: fmt.Errorf("failed to start image build: %w", err)}
			return
		}
		defer img.Body.Close()

		streamLogs, streamResult := streamBuildLogs(img.Body)
		for msg := range streamLogs {
			logs <- msg
		}

		result <- <-streamResult
	}()

	return logs, result
}

func streamBuildLogs(body io.Reader) (<-chan string, <-chan buildResult) {
	logs, result := make(chan string), make(chan buildResult, 1)

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
				result <- buildResult{Err: fmt.Errorf("build stream failed: %w", err)}
				return
			}

			if out.Error != "" {
				result <- buildResult{Err: fmt.Errorf("docker build failed: %s", out.Error)}
				return
			}

			if out.Aux.ID != "" {
				imageID = out.Aux.ID
			}

			if out.Stream != "" {
				logs <- out.Stream
			}
		}

		result <- buildResult{ImageID: imageID}
	}()

	return logs, result
}

func tarDir(srcPath string) (io.ReadCloser, error) {
	pr, pw := io.Pipe()

	go func() {
		tw := tar.NewWriter(pw)

		err := filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}

			// Make paths relative — Docker requires this for the build context
			header.Name, err = filepath.Rel(srcPath, path)
			if err != nil {
				return err
			}
			header.Name = filepath.ToSlash(header.Name)

			if header.Name == "." {
				return nil
			}

			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = io.Copy(tw, f)
			return err
		})

		if err != nil {
			pw.CloseWithError(err)
			return
		}

		pw.CloseWithError(tw.Close())
	}()

	return pr, nil
}
