package queue

import (
	"strings"

	"github.com/HJyup/patchdock/internal/types"
)

type Reporter interface {
	StageChange(stage types.StageName, attempt int)
	StageActivity(activity string)
	StageNote(note string)
}

type reporter struct {
	queue *Queue
	runID string
}

func (r *reporter) StageChange(stage types.StageName, attempt int) {
	msg := stageMsg{
		runID:   r.runID,
		stage:   stage,
		attempt: attempt,
	}

	select {
	case r.queue.inbox <- msg:
	case <-r.queue.ctx.Done():
	}
}

func (r *reporter) StageActivity(activity string) {
	select {
	case r.queue.inbox <- activityMsg{runID: r.runID, text: activity}:
	default:
	}
}

// StageNote arrives as prose and may span lines. Summary is a one-line field,
// the way Activity already is by the time events.go is done with it, so it is
// flattened here rather than left for every client to normalise
func (r *reporter) StageNote(note string) {
	note = strings.TrimSpace(strings.ReplaceAll(note, "\n", " "))
	if note == "" {
		return
	}

	select {
	case r.queue.inbox <- summaryMsg{runID: r.runID, text: note}:
	case <-r.queue.ctx.Done():
	}
}
