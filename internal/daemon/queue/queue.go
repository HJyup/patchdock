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
	Runner    Runner
	Retention time.Duration
}

type run struct {
	state *api.Run
	task  types.Task
}

type Queue struct {
	inbox  chan message
	snaps  chan api.Snapshot
	runner Runner

	// defines retention policy for finilised runs
	retention time.Duration
	ctx       context.Context

	runs map[string]*run
	// define all nesseary context cancel function so it's easy to cancel certain runs
	cancels map[string]context.CancelFunc

	// cloning runs are the most expensive operation in the Queue. Dirty will guard of cloning up-to-date data
	dirty bool
}

func New(cfg Config) *Queue {
	return &Queue{
		inbox:  make(chan message, inboxSize),
		snaps:  make(chan api.Snapshot, 1),
		runner: cfg.Runner,

		retention: cfg.Retention,
		runs:      make(map[string]*run),
		cancels:   make(map[string]context.CancelFunc),
	}
}

func (q *Queue) Run(ctx context.Context) {
	q.ctx = ctx

	ticker := time.NewTicker(publishInterval)
	defer ticker.Stop()
	defer close(q.snaps)

	q.publish()
	for {
		select {
		case <-ctx.Done():
			return

		case msg := <-q.inbox:
			q.handle(msg)

		case <-ticker.C:
			q.evict()

			if q.dirty {
				q.publish()
				q.dirty = false
			}
		}
	}
}

func (q *Queue) Snaps() <-chan api.Snapshot {
	return q.snaps
}

func (q *Queue) Add(ctx context.Context, repo string, task types.Task) (string, error) {
	res := make(chan string, 1)
	select {
	case q.inbox <- addMsg{repo: filepath.Clean(repo), task: task, res: res}:
	case <-ctx.Done():
		return "", ctx.Err()
	}

	select {
	case id := <-res:
		return id, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (q *Queue) Cancel(ctx context.Context, runID string) error {
	reply := make(chan error, 1)

	select {
	case q.inbox <- cancelMsg{runID: runID, err: reply}:
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

func (q *Queue) handle(msg message) {
	switch m := msg.(type) {
	case addMsg:
		q.add(m)
	case cancelMsg:
		q.cancel(m)
	case stageMsg:
		q.stage(m)
	case activityMsg:
		q.activity(m)
	case summaryMsg:
		q.summary(m)
	case doneMsg:
		q.done(m)
	}
}

func (q *Queue) add(m addMsg) {
	ctx, cancel := context.WithCancel(q.ctx)
	id := utils.NewID("run")

	r := &run{
		task: m.task,
		state: &api.Run{
			ID:       id,
			TaskID:   m.task.ID,
			Repo:     m.repo,
			Title:    utils.FirstLine(m.task.Description),
			Status:   api.StatusQueued,
			QueuedAt: time.Now(),
		},
	}

	q.runs[id] = r
	q.dirty = true

	// TODO: This is where the scheduler will come into place
	// Right now it's sequential

	m.res <- id
	q.cancels[r.state.ID] = cancel

	now := time.Now()
	r.state.Status = api.StatusStarted
	r.state.StartedAt = &now
	q.dirty = true

	go q.execute(ctx, RunSpec{
		RunID: r.state.ID,
		Repo:  r.state.Repo,
		Task:  r.task,
	})
}

func (q *Queue) cancel(m cancelMsg) {
	r, ok := q.runs[m.runID]
	if !ok {
		m.err <- ErrNotFound
		return
	}

	if api.IsFinilised(r.state.Status) {
		m.err <- ErrFinished
		return
	}

	if cancel, ok := q.cancels[m.runID]; ok {
		cancel()
	}

	m.err <- nil
}

func (q *Queue) stage(m stageMsg) {
	r, ok := q.runs[m.runID]
	if !ok {
		return
	}

	r.state.Status = api.StatusForStage(m.stage)
	if m.attempt > 0 {
		r.state.Attempt = m.attempt
	}

	now := time.Now()
	r.state.StageStartedAt = &now

	r.state.Activity = ""
	q.dirty = true
}

func (q *Queue) activity(m activityMsg) {
	r, ok := q.runs[m.runID]
	if !ok || api.IsFinilised(r.state.Status) {
		return
	}

	if r.state.Activity == m.text {
		return
	}

	r.state.Activity = m.text
	q.dirty = true
}

func (q *Queue) summary(m summaryMsg) {
	r, ok := q.runs[m.runID]
	if !ok || r.state.Summary == m.text {
		return
	}

	r.state.Summary = m.text
	q.dirty = true
}

func (q *Queue) done(m doneMsg) {
	r, ok := q.runs[m.runID]
	if !ok {
		return
	}

	now := time.Now()
	r.state.FinishedAt = &now
	r.state.Activity = ""

	switch {
	case m.cancelled:
		r.state.Status = api.StatusCancelled
	case m.err != nil:
		r.state.Status = api.StatusFailed
		r.state.Summary = m.err.Error()
	default:
		patch := m.out.Patch
		r.state.Patch = &patch
		if m.out.Accepted {
			r.state.Status = api.StatusSucceeded
			r.state.Branch = m.out.Branch
		} else {
			r.state.Status = api.StatusRejected
		}
	}

	delete(q.cancels, m.runID)
	q.dirty = true
}

func (q *Queue) execute(ctx context.Context, spec RunSpec) {
	out, err := q.runner(ctx, spec, &reporter{queue: q, runID: spec.RunID})
	msg := doneMsg{
		runID:     spec.RunID,
		out:       out,
		err:       err,
		cancelled: ctx.Err() != nil,
	}

	select {
	case q.inbox <- msg:
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
