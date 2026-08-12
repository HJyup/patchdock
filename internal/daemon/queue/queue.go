package queue

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"github.com/HJyup/patchdock/internal/daemon/api"
	"github.com/HJyup/patchdock/internal/daemon/config"
	"github.com/HJyup/patchdock/internal/types"
	"github.com/HJyup/patchdock/internal/utils"
)

var (
	ErrNotFound = errors.New("run not found")
	ErrFinished = errors.New("run has already finished")
)

// run represents a live state published to watchers
type run struct {
	state *api.Run
	task  types.Task
}

// queuedRun is an admission ticket for a run that has not started yet
type queuedRun struct {
	ctx   context.Context
	runID string
}

type Queue struct {
	// Core
	inbox  chan event
	snaps  chan api.Snapshot
	runner Runner

	// Config for making queue work
	retention     time.Duration
	maxContainers int
	snapshotTick  time.Duration
	ctx           context.Context

	// State information
	runs    map[string]*run
	cancels map[string]context.CancelFunc
	dirty   bool

	// Scheduling
	queuedRuns []queuedRun
}

func New(ctx context.Context, runner Runner, cfg *config.Config) *Queue {
	return &Queue{
		inbox:  make(chan event, cfg.InboxSize),
		snaps:  make(chan api.Snapshot, 1),
		runner: runner,

		maxContainers: cfg.MaxContainers,
		snapshotTick:  cfg.SnapshotTick.Duration(),
		retention:     cfg.Retention.Duration(),
		ctx:           ctx,

		runs:    make(map[string]*run),
		cancels: make(map[string]context.CancelFunc),

		queuedRuns: make([]queuedRun, 0),
	}
}

func (q *Queue) Run() {
	ticker := time.NewTicker(q.snapshotTick)
	defer ticker.Stop()

	defer close(q.snaps)

	// Publish empty state to the queue
	q.publish()

	for {
		select {
		case <-q.ctx.Done():
			return

		case event := <-q.inbox:
			q.handle(event)

		case <-ticker.C:
			q.evict()
			q.schedule()

			if q.dirty {
				q.publish()
				q.dirty = false
			}
		}
	}
}

// Snapshots returns a channel as a single point of recieving updates from the queue
func (q *Queue) Snapshots() <-chan api.Snapshot {
	return q.snaps
}

// Core queue functions (remove, schedule, publish)

func (q *Queue) evict() {
	cutoff := time.Now().Add(-q.retention)

	for id, r := range q.runs {
		if r.state.FinishedAt != nil && r.state.FinishedAt.Before(cutoff) {
			delete(q.runs, id)
			q.dirty = true
		}
	}
}

func (q *Queue) publish() {
	snap := q.snapshot()

	utils.SendLatest(q.snaps, snap)
}

func (q *Queue) schedule() {
	for len(q.queuedRuns) > 0 && q.activeCount() < q.maxContainers {
		queued := q.queuedRuns[0]
		q.queuedRuns = q.queuedRuns[1:]

		r, ok := q.runs[queued.runID]
		if !ok || api.IsFinilised(r.state.Status) {
			continue
		}

		// Has been cannceled before (invalidate queued slice)
		if queued.ctx.Err() != nil {
			q.done(doneEvent{runID: queued.runID, cancelled: true})
			continue
		}

		now := time.Now()
		r.state.Status = api.StatusStarted
		r.state.StartedAt = &now
		q.dirty = true

		go q.execute(queued.ctx, RunSpec{
			RunID: r.state.ID,
			Repo:  r.state.Repo,
			Task:  r.task,
		})
	}
}

// Tasks public methods

// Add queues one task and blocks until the queue assigns it a run ID
func (q *Queue) Add(repo string, task types.Task) (string, error) {
	res := make(chan string, 1)

	if !q.send(q.ctx, addEvent{repo: filepath.Clean(repo), task: task, res: res}) {
		return "", q.ctx.Err()
	}

	select {
	case id := <-res:
		return id, nil
	case <-q.ctx.Done():
		return "", q.ctx.Err()
	}
}

// Cancel stops a run and reports whether the queue accepted the request
func (q *Queue) Cancel(ctx context.Context, runID string) error {
	reply := make(chan error, 1)

	if !q.send(ctx, cancelEvent{runID: runID, res: reply}) {
		return ctx.Err()
	}

	select {
	case err := <-reply:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Message handling

func (q *Queue) handle(ev event) {
	switch e := ev.(type) {
	case addEvent:
		q.add(e)
	case cancelEvent:
		q.cancel(e)
	case stageEvent:
		q.stage(e)
	case activityEvent:
		q.activity(e)
	case summaryEvent:
		q.summary(e)
	case doneEvent:
		q.done(e)
	}
}

func (q *Queue) add(e addEvent) {
	ctx, cancel := context.WithCancel(q.ctx)
	id := utils.NewID("run")

	r := &run{
		task: e.task,
		state: &api.Run{
			ID:       id,
			Repo:     e.repo,
			Title:    utils.FirstLine(e.task.Description),
			Status:   api.StatusQueued,
			QueuedAt: time.Now(),
		},
	}

	q.runs[id] = r
	q.dirty = true

	e.res <- id
	q.cancels[r.state.ID] = cancel
	q.queuedRuns = append(q.queuedRuns, queuedRun{ctx: ctx, runID: id})
}

func (q *Queue) cancel(e cancelEvent) {
	r, ok := q.runs[e.runID]
	if !ok {
		e.res <- ErrNotFound
		return
	}

	if api.IsFinilised(r.state.Status) {
		e.res <- ErrFinished
		return
	}

	if cancel, ok := q.cancels[e.runID]; ok {
		cancel()
	}

	// A run that never started has no pipeline to notice the cancelled context
	// and report back, so retire it here. Waiting for the scheduler to do it is
	// not enough: that only happens when a container slot is free, so on a busy
	// queue the run would sit as queued until one opened up. Its ticket stays in
	// queuedRuns and is skipped at admission by the finalised check.
	if r.state.StartedAt == nil {
		q.done(doneEvent{runID: e.runID, cancelled: true})
	}

	e.res <- nil
}

func (q *Queue) stage(e stageEvent) {
	r, ok := q.runs[e.runID]
	if !ok {
		return
	}

	r.state.Status = api.StatusForStage(e.stage)
	if e.attempt > 0 {
		r.state.Attempt = e.attempt
	}

	now := time.Now()
	r.state.StageStartedAt = &now

	r.state.Activity = ""
	q.dirty = true
}

func (q *Queue) activity(e activityEvent) {
	r, ok := q.runs[e.runID]
	if !ok || api.IsFinilised(r.state.Status) {
		return
	}

	if r.state.Activity == e.text {
		return
	}

	r.state.Activity = e.text
	q.dirty = true
}

func (q *Queue) summary(e summaryEvent) {
	r, ok := q.runs[e.runID]
	if !ok || r.state.Summary == e.text {
		return
	}

	r.state.Summary = e.text
	q.dirty = true
}

func (q *Queue) done(e doneEvent) {
	r, ok := q.runs[e.runID]
	if !ok {
		return
	}

	now := time.Now()
	r.state.FinishedAt = &now
	r.state.Activity = ""

	switch {
	case e.cancelled:
		r.state.Status = api.StatusCancelled
	case e.err != nil:
		r.state.Status = api.StatusFailed
		r.state.Summary = e.err.Error()
	default:
		patch := e.out.Patch
		r.state.Patch = &patch
		if e.out.Accepted {
			r.state.Status = api.StatusSucceeded
			r.state.Branch = e.out.Branch
		} else {
			r.state.Status = api.StatusRejected
		}
	}

	if cancel, ok := q.cancels[e.runID]; ok {
		cancel()
		delete(q.cancels, e.runID)
	}

	q.dirty = true
}

// Execution & Additional methods

func (q *Queue) execute(ctx context.Context, spec RunSpec) {
	out, err := q.runner(ctx, spec, &reporter{queue: q, runID: spec.RunID})
	q.send(q.ctx, doneEvent{
		runID:     spec.RunID,
		out:       out,
		err:       err,
		cancelled: ctx.Err() != nil,
	})
}

func (q *Queue) snapshot() api.Snapshot {
	runs := make([]api.Run, 0, len(q.runs))
	for _, r := range q.runs {
		runs = append(runs, r.state.Clone())
	}
	return api.Snapshot{At: time.Now(), Runs: runs}
}

func (q *Queue) activeCount() int {
	n := 0
	for _, r := range q.runs {
		if r.state.StartedAt != nil && !api.IsFinilised(r.state.Status) {
			n++
		}
	}
	return n
}

// send hands event to the queue loop, abandoning the attempt if ctx is cancelled
func (q *Queue) send(ctx context.Context, ev event) bool {
	select {
	case q.inbox <- ev:
		return true
	case <-ctx.Done():
		return false
	}
}
