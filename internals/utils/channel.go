package utils

import "sync"

func FanIn[T any](channels ...<-chan T) <-chan T {
	merged := make(chan T)
	var wg sync.WaitGroup

	for _, ch := range channels {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for msg := range ch {
				merged <- msg
			}
		}()
	}

	go func() {
		wg.Wait()
		close(merged)
	}()

	return merged
}
