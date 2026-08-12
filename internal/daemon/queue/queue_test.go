package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HJyup/patchdock/internal/auditlog"
	"github.com/HJyup/patchdock/internal/daemon/api"
	"github.com/HJyup/patchdock/internal/daemon/config"
	"github.com/HJyup/patchdock/internal/types"
	"github.com/HJyup/patchdock/internal/utils"
)

const testTick = 1 * time.Millisecond
const waitLimit = 3 * time.Second

func testConfig(maxContainers int) config.Config {
	cfg := config.Defaults()
	cfg.MaxContainers = maxContainers
	cfg.SnapshotTick = utils.Duration(testTick)
	return cfg
}

func newQueue(t *testing.T, cfg config.Config) *Queue {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	return New(ctx, nil, &cfg)
}

func newRunningQueue(t *testing.T, cfg config.Config, runner Runner) *Queue {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	q := New(ctx, runner, &cfg)
	go q.Run()

	return q
}

func mustTask(t *testing.T, description string) types.Task {
	t.Helper()

	task, err := types.NewTask(types.Task{Description: description})
	if err != nil {
		t.Fatalf("build task: %v", err)
	}
	return task
}

func seedRun(t *testing.T, q *Queue, description string) string {
	t.Helper()

	res := make(chan string, 1)
	q.add(addEvent{repo: "/repo", task: mustTask(t, description), res: res})

	select {
	case id := <-res:
		return id
	default:
		t.Fatal("add did not reply with a run id")
		return ""
	}
}

func awaitRun(t *testing.T, q *Queue, runID string, want api.Status) api.Run {
	t.Helper()

	deadline := time.After(waitLimit)
	var last api.Status

	for {
		select {
		case snap := <-q.Snapshots():
			for _, r := range snap.Runs {
				if r.ID != runID {
					continue
				}
				last = r.Status
				if r.Status == want {
					return r
				}
			}
		case <-deadline:
			t.Fatalf("run %s reached %q, want %q", runID, last, want)
			return api.Run{}
		}
	}
}

type blockingRunner struct {
	started chan string
	release chan struct{}
}

func newBlockingRunner(capacity int) *blockingRunner {
	return &blockingRunner{
		started: make(chan string, capacity),
		release: make(chan struct{}),
	}
}

func (b *blockingRunner) run(ctx context.Context, spec RunSpec, _ Reporter) (Outcome, error) {
	b.started <- spec.RunID

	select {
	case <-b.release:
	case <-ctx.Done():
	}
	return Outcome{}, nil
}

func (b *blockingRunner) awaitStart(t *testing.T) string {
	t.Helper()

	select {
	case id := <-b.started:
		return id
	case <-time.After(waitLimit):
		t.Fatal("no run was admitted")
		return ""
	}
}

func (b *blockingRunner) assertNoStart(t *testing.T, within time.Duration) {
	t.Helper()

	select {
	case id := <-b.started:
		t.Fatalf("run %s was admitted when it should not have been", id)
	case <-time.After(within):
	}
}

// States

func TestAddRegistersAQueuedRun(t *testing.T) {
	q := newQueue(t, testConfig(1))

	id := seedRun(t, q, "first line\nsecond line")

	r, ok := q.runs[id]
	if !ok {
		t.Fatalf("run %s is not in the run map", id)
	}

	if got := r.state.Status; got != api.StatusQueued {
		t.Errorf("status = %q, want %q", got, api.StatusQueued)
	}
	if got := r.state.Title; got != "first line" {
		t.Errorf("title = %q, want the first line of the description", got)
	}
	if r.state.QueuedAt.IsZero() {
		t.Error("QueuedAt was never stamped")
	}
	if r.state.StartedAt != nil {
		t.Error("a queued run must not have StartedAt")
	}
	if _, ok := q.cancels[id]; !ok {
		t.Error("no cancel func registered, so the run can never be cancelled")
	}
	if len(q.queuedRuns) != 1 || q.queuedRuns[0].runID != id {
		t.Errorf("queuedRuns = %+v, want one ticket for %s", q.queuedRuns, id)
	}
	if !q.dirty {
		t.Error("adding a run must mark the queue dirty so the change is published")
	}
}

func TestCancelUnknownRun(t *testing.T) {
	q := newQueue(t, testConfig(1))

	res := make(chan error, 1)
	q.cancel(cancelEvent{runID: "run-nope", res: res})

	if err := <-res; !errors.Is(err, ErrNotFound) {
		t.Fatalf("cancel(unknown) = %v, want %v", err, ErrNotFound)
	}
}

func TestCancelFinishedRun(t *testing.T) {
	q := newQueue(t, testConfig(1))
	id := seedRun(t, q, "already done")
	q.done(doneEvent{runID: id, out: Outcome{Accepted: true}})

	res := make(chan error, 1)
	q.cancel(cancelEvent{runID: id, res: res})

	if err := <-res; !errors.Is(err, ErrFinished) {
		t.Fatalf("cancel(finished) = %v, want %v", err, ErrFinished)
	}
}

func TestCancelQueuedRunRetiresItImmediately(t *testing.T) {
	q := newQueue(t, testConfig(1))
	id := seedRun(t, q, "never started")
	runCtx := q.queuedRuns[0].ctx

	res := make(chan error, 1)
	q.cancel(cancelEvent{runID: id, res: res})

	if err := <-res; err != nil {
		t.Fatalf("cancel(queued) = %v, want nil", err)
	}
	if runCtx.Err() == nil {
		t.Error("the run's context was not cancelled")
	}

	state := q.runs[id].state
	if state.Status != api.StatusCancelled {
		t.Errorf("status = %q, want %q", state.Status, api.StatusCancelled)
	}
	if state.FinishedAt == nil {
		t.Error("FinishedAt was never stamped, so evict will never reap it")
	}
	if state.StartedAt != nil {
		t.Error("a run cancelled before admission must never have started")
	}
}

func TestCancelRunningRunOnlyTripsTheContext(t *testing.T) {
	q := newQueue(t, testConfig(1))
	id := seedRun(t, q, "in flight")
	runCtx := q.queuedRuns[0].ctx

	now := time.Now()
	q.runs[id].state.StartedAt = &now
	q.runs[id].state.Status = api.StatusCoding

	res := make(chan error, 1)
	q.cancel(cancelEvent{runID: id, res: res})

	if err := <-res; err != nil {
		t.Fatalf("cancel(running) = %v, want nil", err)
	}
	if runCtx.Err() == nil {
		t.Error("the run's context was not cancelled")
	}
	if got := q.runs[id].state.Status; got != api.StatusCoding {
		t.Errorf("status = %q, want it to stay %q until execute reports done", got, api.StatusCoding)
	}
	if q.runs[id].state.FinishedAt != nil {
		t.Error("a still-running run was marked finished before its pipeline reported")
	}
}

func TestStageUpdatesStatusAndAttempt(t *testing.T) {
	tests := []struct {
		stage types.StageName
		want  api.Status
	}{
		{types.StagePlanner, api.StatusPlanning},
		{types.StageExecutor, api.StatusCoding},
		{types.StageReviewer, api.StatusReviewing},
	}

	for _, tt := range tests {
		t.Run(string(tt.stage), func(t *testing.T) {
			q := newQueue(t, testConfig(1))
			id := seedRun(t, q, "work")
			q.runs[id].state.Activity = "stale activity"

			q.stage(stageEvent{runID: id, stage: tt.stage, attempt: 2})

			state := q.runs[id].state
			if state.Status != tt.want {
				t.Errorf("status = %q, want %q", state.Status, tt.want)
			}
			if state.Attempt != 2 {
				t.Errorf("attempt = %d, want 2", state.Attempt)
			}
			if state.StageStartedAt == nil {
				t.Error("StageStartedAt was never stamped")
			}
			if state.Activity != "" {
				t.Errorf("activity = %q, want it cleared on a stage change", state.Activity)
			}
		})
	}
}

func TestSummaryIsIgnoredWhenUnchanged(t *testing.T) {
	q := newQueue(t, testConfig(1))
	id := seedRun(t, q, "work")

	q.summary(summaryEvent{runID: id, text: "a plan"})
	if got := q.runs[id].state.Summary; got != "a plan" {
		t.Fatalf("summary = %q, want %q", got, "a plan")
	}

	q.dirty = false
	q.summary(summaryEvent{runID: id, text: "a plan"})
	if q.dirty {
		t.Error("repeating the same summary marked the queue dirty")
	}
}

func TestDoneTerminalStates(t *testing.T) {
	patch := auditlog.PatchStat{Files: 2, Additions: 10, Deletions: 3}

	tests := []struct {
		name        string
		event       doneEvent
		wantStatus  api.Status
		wantBranch  string
		wantSummary string
		wantPatch   bool
	}{
		{
			name:       "accepted",
			event:      doneEvent{out: Outcome{Accepted: true, Branch: "patchdock/run-1", Patch: patch}},
			wantStatus: api.StatusSucceeded,
			wantBranch: "patchdock/run-1",
			wantPatch:  true,
		},
		{
			name:       "reviewer rejected every attempt",
			event:      doneEvent{out: Outcome{Accepted: false, Patch: patch}},
			wantStatus: api.StatusRejected,
			wantPatch:  true,
		},
		{
			name:        "stage errored",
			event:       doneEvent{err: errors.New("planner stage: boom")},
			wantStatus:  api.StatusFailed,
			wantSummary: "planner stage: boom",
		},
		{
			name:       "cancelled",
			event:      doneEvent{cancelled: true},
			wantStatus: api.StatusCancelled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := newQueue(t, testConfig(1))
			id := seedRun(t, q, "work")
			q.runs[id].state.Activity = "mid-flight activity"

			tt.event.runID = id
			q.done(tt.event)

			state := q.runs[id].state
			if state.Status != tt.wantStatus {
				t.Errorf("status = %q, want %q", state.Status, tt.wantStatus)
			}
			if state.Branch != tt.wantBranch {
				t.Errorf("branch = %q, want %q", state.Branch, tt.wantBranch)
			}
			if tt.wantSummary != "" && state.Summary != tt.wantSummary {
				t.Errorf("summary = %q, want %q", state.Summary, tt.wantSummary)
			}
			if got := state.Patch != nil; got != tt.wantPatch {
				t.Errorf("patch recorded = %v, want %v", got, tt.wantPatch)
			}

			if state.FinishedAt == nil {
				t.Error("FinishedAt was never stamped, so this run will never be evicted")
			}
			if state.Activity != "" {
				t.Errorf("activity = %q, want it cleared on a finished run", state.Activity)
			}
			if !api.IsFinilised(state.Status) {
				t.Errorf("status %q is not treated as finalised", state.Status)
			}
		})
	}
}

func TestDoneReleasesTheCancelFunc(t *testing.T) {
	q := newQueue(t, testConfig(1))
	id := seedRun(t, q, "work")

	q.done(doneEvent{runID: id, out: Outcome{Accepted: true}})

	if _, ok := q.cancels[id]; ok {
		t.Error("cancel func was not released, so the map grows for every run")
	}
}

func TestDoneIgnoresUnknownRuns(t *testing.T) {
	q := newQueue(t, testConfig(1))

	q.done(doneEvent{runID: "run-evicted", out: Outcome{Accepted: true}})

	if len(q.runs) != 0 {
		t.Errorf("runs = %+v, want a done for an unknown run to be dropped", q.runs)
	}
}

func TestEvictReapsOnlyFinishedRunsPastRetention(t *testing.T) {
	cfg := testConfig(1)
	cfg.Retention = utils.Duration(time.Minute)
	q := newQueue(t, cfg)

	stale := seedRun(t, q, "finished long ago")
	recent := seedRun(t, q, "just finished")
	live := seedRun(t, q, "still running")

	q.done(doneEvent{runID: stale, out: Outcome{Accepted: true}})
	q.done(doneEvent{runID: recent, out: Outcome{Accepted: true}})

	// Backdate rather than wait: retention is measured, not slept through.
	old := time.Now().Add(-2 * time.Minute)
	q.runs[stale].state.FinishedAt = &old

	q.evict()

	if _, ok := q.runs[stale]; ok {
		t.Error("a run finished past the retention window was not evicted")
	}
	if _, ok := q.runs[recent]; !ok {
		t.Error("a recently finished run was evicted too early")
	}
	if _, ok := q.runs[live]; !ok {
		t.Error("an unfinished run was evicted")
	}
}

func TestActiveCountOnlyCountsRunningRuns(t *testing.T) {
	q := newQueue(t, testConfig(3))

	queued := seedRun(t, q, "queued")
	running := seedRun(t, q, "running")
	finished := seedRun(t, q, "finished")

	now := time.Now()
	q.runs[running].state.StartedAt = &now
	q.runs[running].state.Status = api.StatusCoding

	q.runs[finished].state.StartedAt = &now
	q.done(doneEvent{runID: finished, out: Outcome{Accepted: true}})

	if got := q.activeCount(); got != 1 {
		t.Fatalf("activeCount = %d, want 1 (queued=%s, finished excluded)", got, queued)
	}
}

func TestSnapshotIsADeepCopy(t *testing.T) {
	q := newQueue(t, testConfig(1))
	id := seedRun(t, q, "work")

	original := time.Now()
	startedAt := original
	q.runs[id].state.StartedAt = &startedAt
	q.runs[id].state.Status = api.StatusCoding

	snap := q.snapshot()
	if len(snap.Runs) != 1 {
		t.Fatalf("snapshot has %d runs, want 1", len(snap.Runs))
	}

	q.runs[id].state.Status = api.StatusSucceeded
	*q.runs[id].state.StartedAt = original.Add(time.Hour)

	if got := snap.Runs[0].Status; got != api.StatusCoding {
		t.Errorf("snapshot status = %q, want it frozen at %q", got, api.StatusCoding)
	}
	if !snap.Runs[0].StartedAt.Equal(original) {
		t.Error("snapshot shares its StartedAt pointer with the live run")
	}
}

// Loop

func TestRunPublishesAnInitialSnapshot(t *testing.T) {
	q := newRunningQueue(t, testConfig(1), nil)

	select {
	case snap := <-q.Snapshots():
		if snap.At.IsZero() {
			t.Error("published a snapshot with no timestamp")
		}
	case <-time.After(waitLimit):
		t.Fatal("no snapshot published: the scheduling loop is not running")
	}
}

func TestScheduleFillsEveryFreeSlot(t *testing.T) {
	runner := newBlockingRunner(3)
	t.Cleanup(func() { close(runner.release) })

	q := newRunningQueue(t, testConfig(3), runner.run)

	want := map[string]bool{}
	for range 3 {
		id, err := q.Add("/repo", mustTask(t, "do the thing"))
		if err != nil {
			t.Fatalf("add run: %v", err)
		}
		want[id] = true
	}

	got := map[string]bool{}
	for range 3 {
		got[runner.awaitStart(t)] = true
	}

	for id := range want {
		if !got[id] {
			t.Errorf("run %s was never admitted", id)
		}
	}
}

func TestScheduleRespectsMaxContainers(t *testing.T) {
	runner := newBlockingRunner(2)
	t.Cleanup(func() { close(runner.release) })

	q := newRunningQueue(t, testConfig(1), runner.run)

	first, err := q.Add("/repo", mustTask(t, "occupies the only slot"))
	if err != nil {
		t.Fatalf("add run: %v", err)
	}
	if _, err := q.Add("/repo", mustTask(t, "must wait")); err != nil {
		t.Fatalf("add run: %v", err)
	}

	if got := runner.awaitStart(t); got != first {
		t.Fatalf("admitted %s first, want %s", got, first)
	}
	runner.assertNoStart(t, 20*testTick)
}

func TestCancelBeforeAdmissionRetiresTheRun(t *testing.T) {
	runner := newBlockingRunner(2)
	t.Cleanup(func() { close(runner.release) })

	q := newRunningQueue(t, testConfig(1), runner.run)

	first, err := q.Add("/repo", mustTask(t, "occupies the only slot"))
	if err != nil {
		t.Fatalf("add run: %v", err)
	}
	queued, err := q.Add("/repo", mustTask(t, "cancelled while waiting"))
	if err != nil {
		t.Fatalf("add run: %v", err)
	}

	if got := runner.awaitStart(t); got != first {
		t.Fatalf("admitted %s first, want %s", got, first)
	}

	if err := q.Cancel(context.Background(), queued); err != nil {
		t.Fatalf("cancel queued run: %v", err)
	}

	run := awaitRun(t, q, queued, api.StatusCancelled)
	if run.FinishedAt == nil {
		t.Error("cancelled run has no FinishedAt, so evict will never reap it")
	}
	if run.StartedAt != nil {
		t.Error("a run cancelled before admission must never have started")
	}
	runner.assertNoStart(t, 20*testTick)
}

func TestRunReachesSucceeded(t *testing.T) {
	outcome := Outcome{
		Accepted: true,
		Branch:   "patchdock/run-1",
		Patch:    auditlog.PatchStat{Files: 1, Additions: 4},
	}

	q := newRunningQueue(t, testConfig(1), func(context.Context, RunSpec, Reporter) (Outcome, error) {
		return outcome, nil
	})

	id, err := q.Add("/repo", mustTask(t, "ship it"))
	if err != nil {
		t.Fatalf("add run: %v", err)
	}

	run := awaitRun(t, q, id, api.StatusSucceeded)
	if run.Branch != outcome.Branch {
		t.Errorf("branch = %q, want %q", run.Branch, outcome.Branch)
	}
	if run.Patch == nil || run.Patch.Files != 1 {
		t.Errorf("patch = %+v, want the runner's stat", run.Patch)
	}
	if run.StartedAt == nil {
		t.Error("a run that executed has no StartedAt")
	}
}

func TestRunReachesFailed(t *testing.T) {
	q := newRunningQueue(t, testConfig(1), func(context.Context, RunSpec, Reporter) (Outcome, error) {
		return Outcome{}, errors.New("planner stage: no such image")
	})

	id, err := q.Add("/repo", mustTask(t, "will fail"))
	if err != nil {
		t.Fatalf("add run: %v", err)
	}

	run := awaitRun(t, q, id, api.StatusFailed)
	if run.Summary != "planner stage: no such image" {
		t.Errorf("summary = %q, want the runner's error", run.Summary)
	}
}

func TestAddCleansTheRepoPath(t *testing.T) {
	runner := newBlockingRunner(1)
	t.Cleanup(func() { close(runner.release) })

	q := newRunningQueue(t, testConfig(1), runner.run)

	id, err := q.Add("/repo/nested/../", mustTask(t, "work"))
	if err != nil {
		t.Fatalf("add run: %v", err)
	}

	runner.awaitStart(t)

	deadline := time.After(waitLimit)
	for {
		select {
		case snap := <-q.Snapshots():
			for _, r := range snap.Runs {
				if r.ID != id {
					continue
				}
				if r.Repo != "/repo" {
					t.Fatalf("repo = %q, want %q", r.Repo, "/repo")
				}
				return
			}
		case <-deadline:
			t.Fatalf("run %s never appeared in a snapshot", id)
		}
	}
}

func TestCancelUnknownRunThroughThePublicAPI(t *testing.T) {
	q := newRunningQueue(t, testConfig(1), nil)

	if err := q.Cancel(context.Background(), "run-does-not-exist"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Cancel(unknown) = %v, want %v", err, ErrNotFound)
	}
}
