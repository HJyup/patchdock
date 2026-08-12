package daemon

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HJyup/patchdock/internal/daemon/broker"
	"github.com/HJyup/patchdock/internal/daemon/config"
	"github.com/HJyup/patchdock/internal/daemon/queue"
	"github.com/HJyup/patchdock/internal/runtimedir"
	"github.com/HJyup/patchdock/internal/utils"
)

func idleRunner(context.Context, queue.RunSpec, queue.Reporter) (queue.Outcome, error) {
	return queue.Outcome{Accepted: true, Branch: "patchdock/run-1"}, nil
}

func blockedRunner(ctx context.Context, _ queue.RunSpec, _ queue.Reporter) (queue.Outcome, error) {
	<-ctx.Done()
	return queue.Outcome{}, ctx.Err()
}

func newTestService(t *testing.T, runner queue.Runner) *Service {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	cfg := config.Defaults()
	cfg.SnapshotTick = utils.Duration(2 * time.Millisecond)

	q := queue.New(ctx, runner, &cfg)
	go q.Run()

	br := broker.New(q.Snapshots())
	go br.Run(ctx)

	dir, err := runtimedir.Resolve(filepath.Join(t.TempDir(), "rt"))
	if err != nil {
		t.Fatalf("resolve runtime dir: %v", err)
	}

	return NewService(q, dir, br)
}

func TestHealthReportsThisProcess(t *testing.T) {
	svc := newTestService(t, idleRunner)

	got := svc.Health(context.Background())

	if got.Status != "ok" {
		t.Errorf("status = %q, want ok", got.Status)
	}

	if got.PID != os.Getpid() {
		t.Errorf("pid = %d, want %d", got.PID, os.Getpid())
	}

	if _, err := time.ParseDuration(got.Uptime); err != nil {
		t.Errorf("uptime %q is not a duration: %v", got.Uptime, err)
	}
}

func TestRunQueuesTheTask(t *testing.T) {
	svc := newTestService(t, blockedRunner)

	resp, err := svc.Run(context.Background(), "/abs/repo", "fix the bug")
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	// Kinda overlaying on the how we represent IDs but used because had some mixed things with ids
	if !strings.HasPrefix(resp.RunID, "run-") {
		t.Errorf("run id = %q, want a run- prefix", resp.RunID)
	}
}

func TestRunRejectsARelativeRepoPath(t *testing.T) {
	svc := newTestService(t, idleRunner)

	_, err := svc.Run(context.Background(), "relative/repo", "fix the bug")
	if !errors.Is(err, ErrInvalidUserPayload) {
		t.Fatalf("error = %v, want it to wrap %v", err, ErrInvalidUserPayload)
	}
}

func TestRunRejectsAnEmptyPrompt(t *testing.T) {
	svc := newTestService(t, idleRunner)

	_, err := svc.Run(context.Background(), "/abs/repo", "")
	if !errors.Is(err, ErrInvalidUserPayload) {
		t.Fatalf("error = %v, want it to wrap %v", err, ErrInvalidUserPayload)
	}
}

func TestCancelQueuedRun(t *testing.T) {
	svc := newTestService(t, blockedRunner)

	resp, err := svc.Run(context.Background(), "/abs/repo", "cancel me")
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	if err := svc.Cancel(context.Background(), resp.RunID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
}

// Stream

func TestSnapshotStreamsQueueState(t *testing.T) {
	svc := newTestService(t, blockedRunner)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	data, errs := svc.Snapshot(ctx)
	resp, err := svc.Run(context.Background(), "/abs/repo", "watch me")
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	deadline := time.After(3 * time.Second)
	for {
		select {
		case snap, ok := <-data:
			if !ok {
				t.Fatal("snapshot stream closed before the run appeared")
			}
			for _, run := range snap.Runs {
				if run.ID == resp.RunID {
					return
				}
			}
		case err := <-errs:
			t.Fatalf("stream error: %v", err)
		case <-deadline:
			t.Fatalf("run %s never appeared in a snapshot", resp.RunID)
		}
	}
}

// Every SSE request cancels its context on disconnect. If Snapshot leaked its
// goroutine or its broker subscription, a busy dashboard would accumulate both.
func TestSnapshotStopsAndClosesOnContextCancel(t *testing.T) {
	svc := newTestService(t, idleRunner)

	ctx, cancel := context.WithCancel(context.Background())
	data, errs := svc.Snapshot(ctx)

	cancel()

	deadline := time.After(3 * time.Second)
	for data != nil || errs != nil {
		select {
		case _, ok := <-data:
			if !ok {
				data = nil
			}
		case _, ok := <-errs:
			if !ok {
				errs = nil
			}
		case <-deadline:
			t.Fatal("Snapshot did not close its channels after its context was cancelled")
		}
	}
}

func TestSnapshotReportsAClosedBroker(t *testing.T) {
	svc := newTestService(t, idleRunner)
	svc.br.Close()

	data, errs := svc.Snapshot(context.Background())

	select {
	case err := <-errs:
		if !errors.Is(err, broker.ErrClosed) {
			t.Fatalf("error = %v, want it to wrap %v", err, broker.ErrClosed)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no error reported for a closed broker")
	}

	select {
	case _, ok := <-data:
		if ok {
			t.Error("a snapshot was delivered despite the broker being closed")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the data channel was never closed")
	}
}
