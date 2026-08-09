package queue

import (
	"github.com/HJyup/patchdock/internal/types"
)

type message interface{ queueMessage() }

type addMsg struct {
	repo string
	task types.Task
	res  chan<- string
}

type cancelMsg struct {
	runID string
	err   chan<- error
}

type stageMsg struct {
	runID   string
	stage   types.StageName
	attempt int
}

type activityMsg struct {
	runID string
	text  string
}

type doneMsg struct {
	runID     string
	out       Outcome
	err       error
	cancelled bool
}

func (addMsg) queueMessage()      {}
func (cancelMsg) queueMessage()   {}
func (stageMsg) queueMessage()    {}
func (activityMsg) queueMessage() {}
func (doneMsg) queueMessage()     {}
