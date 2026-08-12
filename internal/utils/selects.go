package utils

// SendLatest replaces whatever is buffered in ch with v and reports whether v
// was delivered. It never blocks (should use capacity of one for channels)
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
