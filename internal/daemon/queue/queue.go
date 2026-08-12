package queue

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"github.com/HJyup/patchdock/internal/daemon/api"
	"github.com/HJyup/patchdock/internal/types"
	"github.com/HJyup/patchdock/internal/utils"
)

const (
	publishInterval = 200 * time.Millisecond
	inboxSize       = 256
)

var (
	ErrNotFound = errors.New("run not found")
	ErrFinished = errors.New("run has already finished")
	ErrRepoPath = errors.New("repo must be an absolute path")
)

type Config struct {
	Runner        Runner
	Retention     time.Duration
	MaxContainers int
}

type run struct {
	state *api.Run
	task  types.Task
}

type queuedTasks struct {
	ctx   context.Context
	specs RunSpec
}

type Queue struct {
	inbox  chan event
	snaps  chan api.Snapshot
	runner Runner

	// defines retention policy for finilised runs
	retention     time.Duration
	maxContainers int
	ctx           context.Context

	runs map[string]*run
	// define all nesseary context cancel function so it's easy to cancel certain runs
	cancels map[string]context.CancelFunc

	// cloning runs are the most expensive operation in the Queue. Dirty will guard of cloning up-to-date data
	dirty bool

	// scheduler implementation arrays
	waiting []queuedTasks
}

func New(ctx context.Context, cfg Config) *Queue {
	return &Queue{
		inbox:         make(chan event, inboxSize),
		snaps:         make(chan api.Snapshot, 1),
		runner:        cfg.Runner,
		maxContainers: cfg.MaxContainers,

		retention: cfg.Retention,
		ctx:       ctx,
		runs:      make(map[string]*run),
		cancels:   make(map[string]context.CancelFunc),
	}
}

func (q *Queue) Run() {
	ticker := time.NewTicker(publishInterval)
	defer ticker.Stop()
	defer close(q.snaps)

	q.publish()
	for {
		select {
		case <-q.ctx.Done():
			return

		case event := <-q.inbox:
			q.handle(event)

		case <-ticker.C:
			q.evict()

			if q.dirty {
				q.schedule()
				q.publish()
				q.dirty = false
			}
		}
	}
}

func (q *Queue) Snaps() <-chan api.Snapshot {
	return q.snaps
}

func (q *Queue) Add(repo string, task types.Task) (string, error) {
	res := make(chan string, 1)
	select {
	case q.inbox <- addEvent{repo: filepath.Clean(repo), task: task, res: res}:
	case <-q.ctx.Done():
		return "", q.ctx.Err()
	}

	select {
	case id := <-res:
		return id, nil
	case <-q.ctx.Done():
		return "", q.ctx.Err()
	}
}

func (q *Queue) Cancel(ctx context.Context, runID string) error {
	reply := make(chan error, 1)

	select {
	case q.inbox <- cancelEvent{runID: runID, err: reply}:
	case <-ctx.Done():
		return ctx.Err()
	}

	select {
	case err := <-reply:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

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
			TaskID:   e.task.ID,
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

	q.waiting = append(q.waiting, queuedTasks{
		ctx: ctx,
		specs: RunSpec{
			RunID: r.state.ID,
			Repo:  r.state.Repo,
			Task:  r.task,
		}})
}

func (q *Queue) cancel(e cancelEvent) {
	r, ok := q.runs[e.runID]
	if !ok {
		e.err <- ErrNotFound
		return
	}

	if api.IsFinilised(r.state.Status) {
		e.err <- ErrFinished
		return
	}

	if cancel, ok := q.cancels[e.runID]; ok {
		cancel()
	}

	e.err <- nil
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

func (q *Queue) active() int {
	n := 0

	for _, r := range q.runs {
		if r.state.StartedAt != nil && !api.IsFinilised(r.state.Status) {
			n++
		}
	}

	return n
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

func (q *Queue) schedule() {
	if q.dirty && q.active() < q.maxContainers && len(q.waiting) > 0 {
		task := q.waiting[0]
		q.waiting = q.waiting[1:]

		r, ok := q.runs[task.specs.RunID]
		if !ok || api.IsFinilised(r.state.Status) {
			return
		}

		now := time.Now()
		r.state.Status = api.StatusStarted
		r.state.StartedAt = &now
		q.dirty = true

		go q.execute(task.ctx, RunSpec{
			RunID: r.state.ID,
			Repo:  r.state.Repo,
			Task:  r.task,
		})
	}
}

func (q *Queue) execute(ctx context.Context, spec RunSpec) {
	out, err := q.runner(ctx, spec, &reporter{queue: q, runID: spec.RunID})
	event := doneEvent{
		runID:     spec.RunID,
		out:       out,
		err:       err,
		cancelled: ctx.Err() != nil,
	}

	select {
	case q.inbox <- event:
	case <-q.ctx.Done():
	}
}

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

	select {
	case <-q.snaps:
	default:
	}

	q.snaps <- snap
}

func (q *Queue) snapshot() api.Snapshot {
	runs := make([]api.Run, 0, len(q.runs))
	for _, r := range q.runs {
		runs = append(runs, r.state.Clone())
	}
	return api.Snapshot{At: time.Now(), Runs: runs}
}
