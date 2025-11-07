package backoff

import "time"

// LinearBackoff implements a linear backoff strategy.
// The delay increases linearly with each retry: interval * (retry + 1).
type LinearBackoff struct {
	// Interval is the base delay that gets multiplied by retry count
	Interval time.Duration
}

// Next calculates the linear delay for the given retry attempt.
// Formula: interval * (retry + 1)
// Example with interval=100ms: retry 0 → 100ms, retry 1 → 200ms, retry 2 → 300ms
func (l *LinearBackoff) Next(retry int) time.Duration {
	return l.Interval * time.Duration(retry+1)
}

// NewLinearBackoff creates a linear backoff with the specified interval.
func NewLinearBackoff(interval time.Duration) *LinearBackoff {
	return &LinearBackoff{
		Interval: interval,
	}
}
