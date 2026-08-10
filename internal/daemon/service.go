package daemon

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/HJyup/patchdock/internal/daemon/api"
	"github.com/HJyup/patchdock/internal/daemon/broker"
	"github.com/HJyup/patchdock/internal/daemon/queue"
	"github.com/HJyup/patchdock/internal/runtimedir"
)

var Version = "dev"

type Service struct {
	dir     runtimedir.Dir
	queue   *queue.Queue
	started time.Time
	br      *broker.Broker
}

func NewService(q *queue.Queue, dir runtimedir.Dir, br *broker.Broker) *Service {
	return &Service{
		dir:     dir,
		queue:   q,
		started: time.Now(),
		br:      br,
	}
}

func (s *Service) Health(_ context.Context) api.HealthResponse {
	return api.HealthResponse{
		Status: "ok",
		Uptime: time.Since(s.started).Round(time.Second).String(),
		PID:    os.Getpid(),
	}
}

func (s *Service) Snapshot(ctx context.Context) (<-chan api.Snapshot, <-chan error) {
	data := make(chan api.Snapshot)
	errs := make(chan error, 1)

	go func() {
		defer close(data)
		defer close(errs)

		sub, err := s.br.Follow()
		if err != nil {
			select {
			case errs <- fmt.Errorf("follow broker: %w", err):
			case <-ctx.Done():
			}
			return
		}
		defer s.br.Unfollow(sub.ID)

		for {
			select {
			case <-ctx.Done():
				return

			case snap, ok := <-sub.Snapshot:
				if !ok {
					return
				}

				select {
				case data <- snap:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return data, errs
}
