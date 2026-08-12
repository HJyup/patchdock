package utils

// SendLatest replaces whatever is buffered in ch with v and reports whether v
// was delivered. It never blocks.
//
// ch should have capacity 1: at larger capacities this drops the oldest
// buffered value rather than replacing the newest, so a reader still sees stale
// values before it reaches v.
func SendLatest[T any](ch chan T, v T) bool {
	select {
	case <-ch:
	default:
	}

	select {
	case ch <- v:
		return true
	default:
		return false
	}
}
