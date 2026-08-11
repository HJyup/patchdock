package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HJyup/patchdock/internal/daemon/api"
	"github.com/HJyup/patchdock/internal/types"
)

func TestCancelActiveRun(t *testing.T) {
	started := make(chan struct{})
	cancelled := make(chan struct{})

	q := New(Config{
		Runner: func(ctx context.Context, spec RunSpec, rep Reporter) (Outcome, error) {
			close(started)
			<-ctx.Done()
			close(cancelled)
			return Outcome{}, ctx.Err()
		},
		Retention: time.Hour,
	})

	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	go q.Run(ctx)
	waitForInitialSnapshot(t, q)

	runID, err := q.Add("/tmp/repo", types.Task{ID: "task-1", Description: "test task"})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	waitForSignal(t, started, "runner start")

	if err := q.Cancel(context.Background(), runID); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	waitForSignal(t, cancelled, "runner cancellation")

	run := waitForRunStatus(t, q, runID, api.StatusCancelled)
	if run.FinishedAt == nil {
		t.Fatalf("FinishedAt is nil for cancelled run")
	}
}

func TestCancelUnknownRun(t *testing.T) {
	q := New(Config{
		Runner:    func(context.Context, RunSpec, Reporter) (Outcome, error) { return Outcome{}, nil },
		Retention: time.Hour,
	})

	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	go q.Run(ctx)
	waitForInitialSnapshot(t, q)

	if err := q.Cancel(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Cancel error = %v, want %v", err, ErrNotFound)
	}
}

func TestCancelFinishedRun(t *testing.T) {
	q := New(Config{
		Runner:    func(context.Context, RunSpec, Reporter) (Outcome, error) { return Outcome{Accepted: true}, nil },
		Retention: time.Hour,
	})

	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	go q.Run(ctx)
	waitForInitialSnapshot(t, q)

	runID, err := q.Add("/tmp/repo", types.Task{ID: "task-1", Description: "test task"})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	waitForRunStatus(t, q, runID, api.StatusSucceeded)

	if err := q.Cancel(context.Background(), runID); !errors.Is(err, ErrFinished) {
		t.Fatalf("Cancel error = %v, want %v", err, ErrFinished)
	}
}

func waitForInitialSnapshot(t *testing.T, q *Queue) {
	t.Helper()
	select {
	case <-q.Snaps():
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for initial snapshot")
	}
}

func waitForSignal(t *testing.T, ch <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", name)
	}
}

func waitForRunStatus(t *testing.T, q *Queue, runID string, status api.Status) api.Run {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case snap := <-q.Snaps():
			for _, run := range snap.Runs {
				if run.ID == runID && run.Status == status {
					return run
				}
			}
		case <-deadline:
			t.Fatalf("timed out waiting for run %s status %s", runID, status)
		}
	}
}
