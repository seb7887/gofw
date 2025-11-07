package backoff

import "time"

// Backoff defines a strategy for calculating delay between retry attempts.
// Different implementations provide different backoff algorithms (exponential, linear, constant, etc).
type Backoff interface {
	// Next calculates the delay before the next retry attempt.
	// The retry parameter indicates which retry attempt this is (0-indexed).
	// Returns the duration to wait before the next attempt.
	Next(retry int) time.Duration
}
