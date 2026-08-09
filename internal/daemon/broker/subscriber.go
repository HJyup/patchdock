package broker

import (
	"sync"

	"github.com/HJyup/patchdock/internal/daemon/api"
)

type Subscriber struct {
	ID       uint64
	Snapshot chan api.Snapshot
	closed   bool
	mu       sync.Mutex
}

func (s *Subscriber) set(snap api.Snapshot) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return false
	}

	// Drain the channel, since a new snapshot incoming
	select {
	case <-s.Snapshot:
	default:
	}

	// Place the value inside
	select {
	case s.Snapshot <- snap:
		return true
	default:
		return false
	}
}

func (s *Subscriber) close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.closed {
		s.closed = true
		close(s.Snapshot)
	}
}
