package queue

import (
	"strings"

	"github.com/HJyup/patchdock/internal/types"
)

type Reporter interface {
	StageChange(stage types.StageName, attempt int)
	StageActivity(activity string)
	StageSummary(note string)
}

type reporter struct {
	queue *Queue
	runID string
}

func (r *reporter) StageChange(stage types.StageName, attempt int) {
	event := stageEvent{
		runID:   r.runID,
		stage:   stage,
		attempt: attempt,
	}

	select {
	case r.queue.inbox <- event:
	case <-r.queue.ctx.Done():
	}
}

func (r *reporter) StageActivity(activity string) {
	select {
	case r.queue.inbox <- activityEvent{runID: r.runID, text: activity}:
	default:
	}
}

func (r *reporter) StageSummary(summary string) {
	summary = strings.TrimSpace(strings.ReplaceAll(summary, "\n", " "))
	if summary == "" {
		return
	}

	select {
	case r.queue.inbox <- summaryEvent{runID: r.runID, text: summary}:
	case <-r.queue.ctx.Done():
	}
}
