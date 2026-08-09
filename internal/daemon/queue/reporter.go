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

// reporter is a running pipeline's only link to the queue. It posts messages
// and never touches run state: the queue goroutine decides what each one means
type reporter struct {
	queue *Queue
	runID string
}

func (r *reporter) StageStarted(stage types.StageName, attempt int) {
	r.queue.post(stageStartedMsg{
		runID:   r.runID,
		stage:   stage,
		attempt: attempt,
	})
}

// The rest report detail the snapshot has nowhere to put. A run's state is
// whichever stage is running now, which StageStarted already covers, and the
// failure that ends a run arrives with the outcome instead. Dropping them here
// is what keeps a snapshot to current state rather than a history
func (r *reporter) StageFinished(types.StageName, int, string, error) {}
func (r *reporter) StageActivity(string)                              {}
func (r *reporter) StageNote(string)                                  {}
