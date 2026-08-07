package daemon

import (
	"context"
	"log"
)

type Queue struct {
	bus <-chan any
}

func NewQueue(ch <-chan any) *Queue {
	return &Queue{
		bus: ch,
	}
}

func (q *Queue) Run(ctx context.Context) {
	go func() {
		for {
			action := <-q.bus
			logJob(action)
		}
	}()
}

func logJob(action any) {
	log.Printf("queue: received a data: %v", action)
}
