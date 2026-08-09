package queue

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/HJyup/patchdock/internal/daemon/api"
	"github.com/HJyup/patchdock/internal/types"
	"github.com/HJyup/patchdock/internal/utils"
)

const (
	publishInterval = 100 * time.Millisecond
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
	inbox chan message
	snaps chan api.Snapshot

	runner    Runner
	retention time.Duration
	ctx       context.Context

	runs    map[string]*run
	cancels map[string]context.CancelFunc
	dirty   bool
}

func New(cfg Config) *Queue {
	return &Queue{
		inbox:     make(chan message, inboxSize),
		snaps:     make(chan api.Snapshot, 1),
		runner:    cfg.Runner,
		retention: cfg.Retention,
		runs:      make(map[string]*run),
		cancels:   make(map[string]context.CancelFunc),
	}
}

func (q *Queue) Snaps() <-chan api.Snapshot {
	return q.snaps
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

func (q *Queue) Submit(ctx context.Context, req api.SubmitRequest) (string, error) {
	if !filepath.IsAbs(req.Repo) {
		return "", ErrRepoPath
	}

	task, err := types.NewTask(types.Task{Description: req.Prompt})
	if err != nil {
		return "", err
	}

	res := make(chan string, 1)
	select {
	case q.inbox <- submitMsg{repo: filepath.Clean(req.Repo), task: task, res: res}:
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

func (q *Queue) post(msg message) {
	select {
	case q.inbox <- msg:
	case <-q.ctx.Done():
	}
}

func (q *Queue) handle(msg message) {
	switch m := msg.(type) {
	case submitMsg:
		q.submit(m)
	case cancelMsg:
		q.cancel(m)
	case stageStartedMsg:
		q.stageStarted(m)
	case doneMsg:
		q.done(m)
	}
}

func (q *Queue) submit(m submitMsg) {
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

	m.res <- id
	q.launch(r)
}

func (q *Queue) cancel(m cancelMsg) {
	r, ok := q.runs[m.runID]
	if !ok {
		m.err <- ErrNotFound
		return
	}

	if terminal(r.state.Status) {
		m.err <- ErrFinished
		return
	}

	if cancel, ok := q.cancels[m.runID]; ok {
		cancel()
	}

	m.err <- nil
}

func (q *Queue) stageStarted(m stageStartedMsg) {
	r, ok := q.runs[m.runID]
	if !ok {
		return
	}

	r.state.Status = api.StatusForStage(m.stage)
	if m.attempt > 0 {
		r.state.Attempt = m.attempt
	}

	q.dirty = true
}

func (q *Queue) done(m doneMsg) {
	r, ok := q.runs[m.runID]
	if !ok {
		return
	}

	now := time.Now()
	r.state.FinishedAt = &now
	r.state.Attempt = 0

	switch {
	case m.cancelled:
		r.state.Status = api.StatusCancelled
	case m.err != nil:
		r.state.Status = api.StatusFailed
		r.state.Reason = m.err.Error()
	case m.out.Accepted:
		r.state.Status = api.StatusSucceeded
		r.state.Branch = m.out.Branch
	default:
		r.state.Status = api.StatusRejected
	}

	delete(q.cancels, m.runID)

	q.dirty = true
}

func (q *Queue) launch(r *run) {
	ctx, cancel := context.WithCancel(q.ctx)

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

func (q *Queue) execute(ctx context.Context, spec RunSpec) {
	out, err := q.runner(ctx, spec, &reporter{queue: q, runID: spec.RunID})

	if p := recover(); p != nil {
		err = fmt.Errorf("pipeline panic: %v", p)
	}

	q.post(doneMsg{
		runID:     spec.RunID,
		out:       out,
		err:       err,
		cancelled: ctx.Err() != nil,
	})
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

func terminal(s api.Status) bool {
	switch s {
	case api.StatusSucceeded, api.StatusRejected, api.StatusFailed, api.StatusCancelled:
		return true
	default:
		return false
	}
}
