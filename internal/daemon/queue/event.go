package queue

import (
	"github.com/HJyup/patchdock/internal/types"
)

// event is the central communication object with a queue.
type event interface{ queueEvent() }

// addEvent represents a new task that needs to be accepted to the queue
type addEvent struct {
	repo string
	task types.Task
	res  chan<- string
}

// cancelEvent represents a queued task (by id) which we want to cancel
type cancelEvent struct {
	runID string
	res   chan<- error
}

// stageEvent changes the state of the run
type stageEvent struct {
	runID   string
	stage   types.StageName
	attempt int
}

// acitivityEvent changes the text of the one-line activity per run
type activityEvent struct {
	runID string
	text  string
}

// summaryEvent represents a summary defined by agents (in current implementation used by planner)
type summaryEvent struct {
	runID string
	text  string
}

// doneEvent represents a full finished run from runner and we want to report back outcome
type doneEvent struct {
	runID     string
	out       Outcome
	err       error
	cancelled bool
}

func (addEvent) queueEvent()      {}
func (cancelEvent) queueEvent()   {}
func (stageEvent) queueEvent()    {}
func (activityEvent) queueEvent() {}
func (summaryEvent) queueEvent()  {}
func (doneEvent) queueEvent()     {}
