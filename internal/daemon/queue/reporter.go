package queue

import (
	"github.com/HJyup/patchdock/internal/types"
)

type Reporter interface {
	StageStarted(stage types.StageName, attempt int)
	StageActivity(activity string)
	StageNote(note string)
	StageFinished(stage types.StageName, attempt int, note string, err error)
}

type reporter struct {
	queue *Queue
	runID string
}

func (r *reporter) StageStarted(stage types.StageName, attempt int) {
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

func (r *reporter) StageFinished(types.StageName, int, string, error) {}
func (r *reporter) StageNote(string)                                  {}
