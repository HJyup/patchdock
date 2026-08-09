package queue

import (
	"github.com/HJyup/patchdock/internal/types"
)

type message interface{ queueMessage() }

type submitMsg struct {
	repo string
	task types.Task
	res  chan<- string
}

type cancelMsg struct {
	runID string
	err   chan<- error
}

type stageStartedMsg struct {
	runID   string
	stage   types.StageName
	attempt int
}

type doneMsg struct {
	runID     string
	out       Outcome
	err       error
	cancelled bool
}

func (submitMsg) queueMessage()       {}
func (cancelMsg) queueMessage()       {}
func (stageStartedMsg) queueMessage() {}
func (doneMsg) queueMessage()         {}
