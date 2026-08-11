package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HJyup/patchdock/internal/auditlog"
	"github.com/HJyup/patchdock/internal/daemon/api"
	"github.com/HJyup/patchdock/internal/types"
)

func TestAddBeforeRunAfterContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	q := New(ctx, Config{
		Runner:    successfulRunner,
		Retention: time.Minute,
	})

	cancel()

	_, err := q.Add(t.TempDir(), types.Task{
		ID:          "task-cancelled",
		Description: "cancelled before run",
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Add() error = %v, want %v", err, context.Canceled)
	}
}

func TestQueueLifecycleUsesConstructorContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reported := make(chan struct{})
	finish := make(chan struct{})
	q := New(ctx, Config{
		Runner: func(ctx context.Context, spec RunSpec, rep Reporter) (Outcome, error) {
			rep.StageChange(types.StagePlanner, 1)
			rep.StageActivity("fake activity")
			rep.StageSummary(" fake\nsummary ")
			close(reported)

			select {
			case <-finish:
			case <-ctx.Done():
				return Outcome{}, ctx.Err()
			}

			return Outcome{
				Accepted: true,
				Branch:   "patchdock/fake-run",
				Patch: auditlog.PatchStat{
					Files:     1,
					Additions: 2,
					Deletions: 1,
				},
			}, nil
		},
		Retention: time.Minute,
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		q.Run()
	}()

	runID, err := q.Add(t.TempDir(), types.Task{
		ID:          "task-lifecycle",
		Description: "exercise fake runner",
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	select {
	case <-reported:
	case <-time.After(time.Second):
		t.Fatal("runner did not report stage updates")
	}

	run := waitForRun(t, q.Snaps(), runID, func(run api.Run) bool {
		return run.Status == api.StatusPlanning && run.Activity == "fake activity" && run.Summary == "fake summary"
	})
	if run.Attempt != 1 {
		t.Fatalf("Attempt = %d, want 1", run.Attempt)
	}

	close(finish)

	run = waitForRunStatus(t, q.Snaps(), runID, api.StatusSucceeded)
	if run.TaskID != "task-lifecycle" {
		t.Fatalf("TaskID = %q, want %q", run.TaskID, "task-lifecycle")
	}
	if run.Attempt != 1 {
		t.Fatalf("Attempt = %d, want 1", run.Attempt)
	}
	if run.Summary != "fake summary" {
		t.Fatalf("Summary = %q, want %q", run.Summary, "fake summary")
	}
	if run.Activity != "" {
		t.Fatalf("Activity = %q, want empty after successful completion", run.Activity)
	}
	if run.Branch != "patchdock/fake-run" {
		t.Fatalf("Branch = %q, want %q", run.Branch, "patchdock/fake-run")
	}
	if run.Patch == nil || *run.Patch != (auditlog.PatchStat{Files: 1, Additions: 2, Deletions: 1}) {
		t.Fatalf("Patch = %#v, want fake patch stats", run.Patch)
	}

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run() did not exit after context cancellation")
	}
}

func successfulRunner(ctx context.Context, spec RunSpec, rep Reporter) (Outcome, error) {
	rep.StageChange(types.StagePlanner, 1)
	rep.StageActivity("fake activity")
	rep.StageSummary(" fake\nsummary ")

	return Outcome{
		Accepted: true,
		Branch:   "patchdock/fake-run",
		Patch: auditlog.PatchStat{
			Files:     1,
			Additions: 2,
			Deletions: 1,
		},
	}, ctx.Err()
}

func waitForRunStatus(t *testing.T, snaps <-chan api.Snapshot, runID string, status api.Status) api.Run {
	t.Helper()

	return waitForRun(t, snaps, runID, func(run api.Run) bool {
		return run.Status == status
	})
}

func waitForRun(t *testing.T, snaps <-chan api.Snapshot, runID string, match func(api.Run) bool) api.Run {
	t.Helper()

	timeout := time.After(3 * time.Second)
	for {
		select {
		case snap, ok := <-snaps:
			if !ok {
				t.Fatalf("snapshots closed before run %s matched expected state", runID)
			}

			for _, run := range snap.Runs {
				if run.ID == runID && match(run) {
					return run
				}
			}

		case <-timeout:
			t.Fatalf("timed out waiting for run %s to match expected state", runID)
		}
	}
}
