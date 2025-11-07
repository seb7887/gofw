package backoff

import "time"

// ConstantBackoff implements a constant backoff strategy.
// The delay remains the same for all retry attempts.
type ConstantBackoff struct {
	// Interval is the fixed delay between retries
	Interval time.Duration
}

// Next returns the constant delay, regardless of retry count.
func (c *ConstantBackoff) Next(retry int) time.Duration {
	return c.Interval
}

// NewConstantBackoff creates a constant backoff with the specified interval.
func NewConstantBackoff(interval time.Duration) *ConstantBackoff {
	return &ConstantBackoff{
		Interval: interval,
	}
}
