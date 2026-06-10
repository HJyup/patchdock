package docker

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
)

const testRunImage = "patchdock-test-run:base"

// ensureRunImage builds the tiny image Run tests execute. The daemon's
// layer cache makes every call after the first effectively free.
func ensureRunImage(t *testing.T, c *Client) {
	t.Helper()
	ctx := context.Background()

	if ok, _ := c.ImageExists(ctx, testRunImage); ok {
		return
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM alpine:3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	logs, res := c.Build(ctx, BuildSpec{ContextDir: dir, Tag: testRunImage})
	for range logs {
	}
	if r := <-res; r.Err != nil {
		t.Fatalf("building test image: %v", r.Err)
	}
}

// collect drains both channels and returns everything.
func collect(logs <-chan LogLine, res <-chan Result) ([]LogLine, Result) {
	var lines []LogLine
	for l := range logs {
		lines = append(lines, l)
	}
	return lines, <-res
}

func TestRunStreamsLogsEnvAndExitCode(t *testing.T) {
	c := newTestClient(t)
	ensureRunImage(t, c)

	logs, res := c.Run(context.Background(), RunSpec{
		Image:      testRunImage,
		Env:        map[string]string{"PD_FOO": "bar"},
		Entrypoint: []string{"sh", "-c", `echo "value=$PD_FOO"; echo oops 1>&2; exit 7`},
	})
	lines, r := collect(logs, res)

	if r.Err != nil {
		t.Fatalf("unexpected error: %v", r.Err)
	}
	if r.ExitCode != 7 {
		t.Errorf("ExitCode = %d, want 7", r.ExitCode)
	}

	var sawStdout, sawStderr bool
	for _, l := range lines {
		if l.Stream == "stdout" && strings.Contains(l.Text, "value=bar") {
			sawStdout = true // env reached the process, demuxed as stdout
		}
		if l.Stream == "stderr" && strings.Contains(l.Text, "oops") {
			sawStderr = true
		}
	}
	if !sawStdout {
		t.Errorf("missing stdout line with env value; got %v", lines)
	}
	if !sawStderr {
		t.Errorf("missing stderr line; got %v", lines)
	}
}

func TestRunMountRoundTrip(t *testing.T) {
	c := newTestClient(t)
	ensureRunImage(t, c)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "in.txt"), []byte("ping"), 0o644); err != nil {
		t.Fatal(err)
	}

	logs, res := c.Run(context.Background(), RunSpec{
		Image:      testRunImage,
		Mounts:     []Mount{{Source: dir, Target: "/data"}},
		Entrypoint: []string{"sh", "-c", "cat /data/in.txt > /data/out.txt"},
	})
	_, r := collect(logs, res)
	if r.Err != nil || r.ExitCode != 0 {
		t.Fatalf("run failed: err=%v exit=%d", r.Err, r.ExitCode)
	}

	// The write inside the container must be visible on the host: this is
	// the exchange mechanism the whole agentio protocol stands on.
	got, err := os.ReadFile(filepath.Join(dir, "out.txt"))
	if err != nil {
		t.Fatalf("container write did not land on host: %v", err)
	}
	if string(got) != "ping" {
		t.Errorf("out.txt = %q, want %q", got, "ping")
	}
}

func TestRunReadOnlyMountBlocksWrites(t *testing.T) {
	c := newTestClient(t)
	ensureRunImage(t, c)

	dir := t.TempDir()

	logs, res := c.Run(context.Background(), RunSpec{
		Image:      testRunImage,
		Mounts:     []Mount{{Source: dir, Target: "/data", ReadOnly: true}},
		Entrypoint: []string{"sh", "-c", "touch /data/x"},
	})
	_, r := collect(logs, res)

	if r.Err != nil {
		t.Fatalf("unexpected transport error: %v", r.Err)
	}
	if r.ExitCode == 0 {
		t.Fatal("write to a read-only mount succeeded; the wall has a hole")
	}
	if _, err := os.Stat(filepath.Join(dir, "x")); err == nil {
		t.Fatal("file appeared on host through a read-only mount")
	}
}

func TestRunRejectsBadMounts(t *testing.T) {
	c := newTestClient(t)

	cases := map[string]Mount{
		"relative path": {Source: "relative/dir", Target: "/data"},
		"missing dir":   {Source: "/no/such/dir/patchdock-test", Target: "/data"},
	}
	for name, m := range cases {
		t.Run(name, func(t *testing.T) {
			logs, res := c.Run(context.Background(), RunSpec{Image: testRunImage, Mounts: []Mount{m}})
			_, r := collect(logs, res)
			if r.Err == nil {
				t.Fatal("expected a validation error, got success")
			}
			if !strings.Contains(r.Err.Error(), "mount source") {
				t.Errorf("error should name the mount source: %v", r.Err)
			}
		})
	}
}

func TestRunCancelKillsLabeledContainerAndRemovesIt(t *testing.T) {
	c := newTestClient(t)
	ensureRunImage(t, c)

	const label = "patchdock.test-cancel"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logs, res := c.Run(ctx, RunSpec{
		Image:      testRunImage,
		Labels:     map[string]string{label: "1"},
		Entrypoint: []string{"sleep", "30"},
	})

	findByLabel := func() int {
		list, err := c.cli.ContainerList(context.Background(), container.ListOptions{
			All:     true,
			Filters: filters.NewArgs(filters.Arg("label", label)),
		})
		if err != nil {
			t.Fatal(err)
		}
		return len(list)
	}

	// While running, the label must make the container findable — this is
	// the handle `patchdock clean` and the crash-reaper depend on.
	deadline := time.Now().Add(10 * time.Second)
	for findByLabel() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("labeled container never appeared")
		}
		time.Sleep(100 * time.Millisecond)
	}

	cancel()
	lines, r := collect(logs, res) // channels must still close cleanly
	_ = lines

	if r.Err == nil {
		t.Fatal("expected a cancellation error, got success")
	}

	// The container must be gone: killed AND removed despite the dead ctx.
	deadline = time.Now().Add(10 * time.Second)
	for findByLabel() != 0 {
		if time.Now().After(deadline) {
			t.Fatal("container leaked after cancellation")
		}
		time.Sleep(100 * time.Millisecond)
	}
}
