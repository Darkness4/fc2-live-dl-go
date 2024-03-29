// Package channel describes functions for working with channels.
package channel

import "time"

// Debounce returns a channel that debounces the input channel.
func Debounce[T any](events <-chan T, duration time.Duration) <-chan T {
	out := make(chan T)
	go func() {
		timer := time.NewTimer(duration)
		var last T
		for {
			select {
			case event, ok := <-events:
				if !ok {
					close(out)
					return
				}
				last = event
				timer.Reset(duration)
			case <-timer.C:
				out <- last
			}
		}
	}()
	return out
}
