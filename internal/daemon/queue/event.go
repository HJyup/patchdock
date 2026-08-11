package queue

import (
	"github.com/HJyup/patchdock/internal/types"
)

type event interface{ queueEvent() }

type addEvent struct {
	repo string
	task types.Task
	res  chan<- string
}

type cancelEvent struct {
	runID string
	err   chan<- error
}

type stageEvent struct {
	runID   string
	stage   types.StageName
	attempt int
}

type activityEvent struct {
	runID string
	text  string
}

type summaryEvent struct {
	runID string
	text  string
}

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
