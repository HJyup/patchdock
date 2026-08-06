package daemon

import (
	"context"
	"os"
	"time"

	"github.com/HJyup/patchdock/internal/daemon/api"
	"github.com/HJyup/patchdock/internal/runtimedir"
)

var Version = "dev"

type Service struct {
	dir     runtimedir.Dir
	queue   chan<- any
	started time.Time
}

func NewService(queue chan<- any, dir runtimedir.Dir) *Service {
	return &Service{
		dir:     dir,
		queue:   queue,
		started: time.Now(),
	}
}

func (s *Service) Queue(_ context.Context, action any) {
	s.queue <- action
}

func (s *Service) Health(_ context.Context) api.HealthResponse {
	return api.HealthResponse{
		Status: "ok",
		Uptime: time.Since(s.started).Round(time.Second).String(),
		PID:    os.Getpid(),
	}
}
