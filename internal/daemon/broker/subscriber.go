package broker

import (
	"sync"

	"github.com/HJyup/patchdock/internal/daemon/api"
	"github.com/HJyup/patchdock/internal/utils"
)

// Subscriber represents a one-to-one connection to broker for a client
type subscriber struct {
	ID       uint64
	Snapshot chan api.Snapshot
	closed   bool
	mu       sync.Mutex
}

func (s *subscriber) set(snap api.Snapshot) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return false
	}

	return utils.SendLatest(s.Snapshot, snap)
}

func (s *subscriber) close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.closed {
		s.closed = true
		close(s.Snapshot)
	}
}
