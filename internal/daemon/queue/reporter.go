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
	r.queue.send(r.queue.ctx, stageEvent{
		runID:   r.runID,
		stage:   stage,
		attempt: attempt,
	})
}

// StageActivity deliberately does not use send: activity text is best-effort
// telemetry for the dashboard, so a full inbox drops it rather than making the
// pipeline wait on the queue loop.
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

	r.queue.send(r.queue.ctx, summaryEvent{runID: r.runID, text: summary})
}
