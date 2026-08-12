package broker

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/HJyup/patchdock/internal/daemon/api"
)

var ErrClosed = errors.New("the broker has been closed")

// Broker centralised places for fan-out all snapshots to clients
type Broker struct {
	// Core
	mu     sync.Mutex
	subs   map[uint64]*subscriber
	snaps  <-chan api.Snapshot
	closed bool

	// Subscribers management
	currID uint64

	// State handling
	lastSnap *api.Snapshot
}

func New(snaps <-chan api.Snapshot) *Broker {
	return &Broker{
		mu:    sync.Mutex{},
		subs:  make(map[uint64]*subscriber),
		snaps: snaps,
	}
}

func (b *Broker) Run(ctx context.Context) {
	defer b.Close()

	for {
		select {
		case <-ctx.Done():
			log.Println("broker: shutting down")
			return

		case snap, ok := <-b.snaps:
			if !ok {
				log.Println("broker: snapshot channel closed")
				return
			}
			b.fanout(snap)
		}
	}
}

// Close removes all connections to the broker subsequently closing all subscribers
func (b *Broker) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.closed = true
	for _, sub := range b.subs {
		sub.close()
	}

	clear(b.subs)
}

// Follow creates a subscriber with unique id for the client
func (b *Broker) Follow() (*subscriber, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil, ErrClosed
	}

	b.currID++
	sub := &subscriber{
		ID:       b.currID,
		Snapshot: make(chan api.Snapshot, 1),
	}

	b.subs[sub.ID] = sub
	if b.lastSnap != nil {
		sub.set(*b.lastSnap)
	}

	return sub, nil
}

// Unfollow removes subscriber for a broker
func (b *Broker) Unfollow(id uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}

	if _, ok := b.subs[id]; !ok {
		return
	}

	b.subs[id].close()
	delete(b.subs, id)
}

func (b *Broker) fanout(snap api.Snapshot) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, sub := range b.subs {
		sub.set(snap)
	}

	b.lastSnap = &snap
}
