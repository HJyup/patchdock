package broker

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/HJyup/patchdock/internal/daemon/api"
)

var ErrClosed = errors.New("the broker has been closed")

// Defines main state where we fannin out snapshot of runs to the subscribers
type Broker struct {
	mu     sync.Mutex
	subs   map[uint64]*Subscriber
	snaps  <-chan api.Snapshot
	currID uint64

	last   *api.Snapshot
	closed bool
}

func New(snaps <-chan api.Snapshot) *Broker {
	return &Broker{
		mu:    sync.Mutex{},
		subs:  make(map[uint64]*Subscriber),
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

func (b *Broker) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.closed = true
	for _, sub := range b.subs {
		sub.close()
	}

	clear(b.subs)
}

func (b *Broker) Follow() (*Subscriber, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil, ErrClosed
	}

	b.currID++
	sub := &Subscriber{
		ID:       b.currID,
		Snapshot: make(chan api.Snapshot, 1),
	}

	b.subs[sub.ID] = sub
	if b.last != nil {
		sub.set(*b.last)
	}

	return sub, nil
}

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

	b.last = &snap
}
