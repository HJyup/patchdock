package daemon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/HJyup/patchdock/internal/daemon/api"
	"github.com/HJyup/patchdock/internal/daemon/broker"
	"github.com/HJyup/patchdock/internal/daemon/queue"
	"github.com/HJyup/patchdock/internal/runtimedir"
	"github.com/HJyup/patchdock/internal/types"
)

var Version = "dev"

type Service struct {
	dir     runtimedir.Dir
	queue   *queue.Queue
	started time.Time
	br      *broker.Broker
}

var ErrInvalidUserPayload = errors.New("invalid user payload")

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

func (s *Service) Run(ctx context.Context, repo string, prompt string) (api.RunResponse, error) {
	if !filepath.IsAbs(repo) {
		return api.RunResponse{}, fmt.Errorf("%w: repo path is not absolute", ErrInvalidUserPayload)
	}

	task, err := types.NewTask(types.Task{Description: prompt})
	if err != nil {
		return api.RunResponse{}, errors.New("failed to create a task")
	}

	id, err := s.queue.Add(repo, task)
	if err != nil {
		return api.RunResponse{}, errors.New("failed to add task to the queue")
	}

	return api.RunResponse{RunID: id}, nil
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
