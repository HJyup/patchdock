package broker

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/HJyup/patchdock/internal/daemon/api"
)

const waitLimit = 3 * time.Second

func snapshotWith(runIDs ...string) api.Snapshot {
	runs := make([]api.Run, 0, len(runIDs))
	for _, id := range runIDs {
		runs = append(runs, api.Run{ID: id, Status: api.StatusQueued})
	}
	return api.Snapshot{At: time.Now(), Runs: runs}
}

func firstRunID(snap api.Snapshot) string {
	if len(snap.Runs) == 0 {
		return ""
	}
	return snap.Runs[0].ID
}

func receive(t *testing.T, sub *subscriber) api.Snapshot {
	t.Helper()

	select {
	case snap, ok := <-sub.Snapshot:
		if !ok {
			t.Fatal("subscriber channel was closed, want a snapshot")
		}
		return snap
	case <-time.After(waitLimit):
		t.Fatal("no snapshot delivered")
		return api.Snapshot{}
	}
}

func assertNoDelivery(t *testing.T, sub *subscriber, within time.Duration) {
	t.Helper()

	select {
	case snap, ok := <-sub.Snapshot:
		if ok {
			t.Fatalf("received %v, want no delivery", firstRunID(snap))
		}
	case <-time.After(within):
	}
}

func awaitClosed(t *testing.T, sub *subscriber) {
	t.Helper()

	deadline := time.After(waitLimit)
	for {
		select {
		case _, ok := <-sub.Snapshot:
			if !ok {
				return
			}
		case <-deadline:
			t.Fatal("subscriber channel was never closed")
		}
	}
}

func mustFollow(t *testing.T, b *Broker) *subscriber {
	t.Helper()

	sub, err := b.Follow()
	if err != nil {
		t.Fatalf("follow: %v", err)
	}
	return sub
}

// Fan-out & subscribers

func TestFanoutReachesEverySubscriber(t *testing.T) {
	b := New(nil)

	first := mustFollow(t, b)
	second := mustFollow(t, b)

	if first.ID == second.ID {
		t.Errorf("subscriber IDs collided at %d", first.ID)
	}

	b.fanout(snapshotWith("run-1"))

	if got := firstRunID(receive(t, first)); got != "run-1" {
		t.Errorf("first subscriber got %q, want run-1", got)
	}
	if got := firstRunID(receive(t, second)); got != "run-1" {
		t.Errorf("second subscriber got %q, want run-1", got)
	}
}

func TestFollowReplaysTheLastSnapshot(t *testing.T) {
	b := New(nil)

	b.fanout(snapshotWith("run-1"))
	b.fanout(snapshotWith("run-2"))

	late := mustFollow(t, b)
	if got := firstRunID(receive(t, late)); got != "run-2" {
		t.Errorf("late subscriber got %q, want the most recent snapshot run-2", got)
	}
}

func TestFollowBeforeAnySnapshotDeliversNothing(t *testing.T) {
	b := New(nil)

	sub := mustFollow(t, b)
	assertNoDelivery(t, sub, 50*time.Millisecond)
}

func TestSlowSubscriberOnlySeesTheLatestSnapshot(t *testing.T) {
	b := New(nil)
	sub := mustFollow(t, b)

	b.fanout(snapshotWith("run-1"))
	b.fanout(snapshotWith("run-2"))
	b.fanout(snapshotWith("run-3"))

	if got := firstRunID(receive(t, sub)); got != "run-3" {
		t.Errorf("got %q, want the newest snapshot run-3", got)
	}

	assertNoDelivery(t, sub, 50*time.Millisecond)
}

func TestUnfollowClosesTheChannelAndStopsDelivery(t *testing.T) {
	b := New(nil)

	staying := mustFollow(t, b)
	leaving := mustFollow(t, b)

	b.Unfollow(leaving.ID)
	awaitClosed(t, leaving)

	b.fanout(snapshotWith("run-1"))
	if got := firstRunID(receive(t, staying)); got != "run-1" {
		t.Errorf("remaining subscriber got %q, want run-1", got)
	}
}

// Shutdown

func TestCloseClosesEverySubscriber(t *testing.T) {
	b := New(nil)

	first := mustFollow(t, b)
	second := mustFollow(t, b)

	b.Close()

	awaitClosed(t, first)
	awaitClosed(t, second)
}

func TestFollowAfterCloseReturnsErrClosed(t *testing.T) {
	b := New(nil)
	b.Close()

	sub, err := b.Follow()
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("Follow after Close = %v, want %v", err, ErrClosed)
	}
	if sub != nil {
		t.Error("Follow returned a subscriber alongside an error")
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	b := New(nil)
	sub := mustFollow(t, b)

	b.Close()
	b.Close() // must not panic

	awaitClosed(t, sub)
}

func TestUnfollowAfterCloseIsANoop(t *testing.T) {
	b := New(nil)
	sub := mustFollow(t, b)

	b.Close()
	b.Unfollow(sub.ID) // must not double-close

	awaitClosed(t, sub)
}

// Run loop

func TestRunFansOutFromItsSource(t *testing.T) {
	source := make(chan api.Snapshot)
	b := New(source)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go b.Run(ctx)

	sub := mustFollow(t, b)
	source <- snapshotWith("run-1")

	if got := firstRunID(receive(t, sub)); got != "run-1" {
		t.Errorf("got %q, want run-1", got)
	}
}

func TestRunClosesSubscribersOnContextCancel(t *testing.T) {
	source := make(chan api.Snapshot)
	b := New(source)

	ctx, cancel := context.WithCancel(context.Background())
	go b.Run(ctx)

	sub := mustFollow(t, b)
	cancel()

	awaitClosed(t, sub)
	if _, err := b.Follow(); !errors.Is(err, ErrClosed) {
		t.Fatalf("Follow after shutdown = %v, want %v", err, ErrClosed)
	}
}

func TestConcurrentFollowUnfollowAndFanout(t *testing.T) {
	source := make(chan api.Snapshot)
	b := New(source)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go b.Run(ctx)

	stop := make(chan struct{})
	producerDone := make(chan struct{})
	go func() {
		defer close(producerDone)
		for {
			select {
			case <-stop:
				return
			case source <- snapshotWith("run-1"):
			}
		}
	}()

	var followers sync.WaitGroup
	for range 8 {
		followers.Go(func() {
			for range 20 {
				sub, err := b.Follow()
				if err != nil {
					return
				}
				select {
				case <-sub.Snapshot:
				default:
				}
				b.Unfollow(sub.ID)
			}
		})
	}

	followers.Wait()

	close(stop)
	<-producerDone
}
