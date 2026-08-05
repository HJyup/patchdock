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
	started time.Time
}

func NewService(dir runtimedir.Dir) *Service {
	return &Service{
		dir:     dir,
		started: time.Now(),
	}
}

func (s *Service) Health(_ context.Context) api.HealthResponse {
	return api.HealthResponse{
		Status: "ok",
		Uptime: time.Since(s.started).Round(time.Second).String(),
		PID:    os.Getpid(),
	}
}
