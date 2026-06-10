package docker

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/docker/api/types/image"
)

// newTestClient skips the test when no Docker daemon is reachable, so the
// suite stays green on machines without Docker. In CI, Docker is
// guaranteed, so an unreachable daemon is a failure — never a silent skip.
func newTestClient(t *testing.T) *Client {
	t.Helper()
	c, err := NewClient()
	if err != nil {
		skipOrFail(t, "docker not available: %v", err)
	}
	t.Cleanup(func() {
		c.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := c.cli.Ping(ctx); err != nil {
		skipOrFail(t, "docker daemon not reachable: %v", err)
	}
	return c
}

func skipOrFail(t *testing.T, format string, args ...any) {
	t.Helper()
	if os.Getenv("CI") != "" {
		t.Fatalf("docker required in CI: "+format, args...)
	}
	t.Skipf(format, args...)
}

func TestBuildTagsImage(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	dir := t.TempDir()
	dockerfile := "FROM alpine:3\nRUN echo patchdock-build-test\n"
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte(dockerfile), 0o644); err != nil {
		t.Fatal(err)
	}

	const tag = "patchdock-test-build:tagcheck"
	t.Cleanup(func() {
		_, _ = c.cli.ImageRemove(ctx, tag, image.RemoveOptions{Force: true})
	})

	logs, res := c.Build(ctx, BuildSpec{ContextDir: dir, Tag: tag})
	for range logs {
		// drain — callers must consume the stream
	}
	r := <-res
	if r.Err != nil {
		t.Fatalf("build failed: %v", r.Err)
	}
	if r.ImageID == "" {
		t.Error("expected a non-empty ImageID from the aux stream")
	}

	// The point of the test: the tag must be findable on the daemon.
	found, err := c.ImageExists(ctx, tag)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatalf("ImageExists(%q) = false after a tagged build", tag)
	}
	if found, _ := c.ImageExists(ctx, "patchdock-test-build:no-such-tag"); found {
		t.Fatal("ImageExists returned true for a tag that was never built")
	}
}

func TestBuildReportsDockerfileErrors(t *testing.T) {
	c := newTestClient(t)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM alpine:3\nRUN exit 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	logs, res := c.Build(context.Background(), BuildSpec{ContextDir: dir})
	for range logs {
	}
	if r := <-res; r.Err == nil {
		t.Fatal("expected an error for a failing RUN step, got nil")
	}
}
