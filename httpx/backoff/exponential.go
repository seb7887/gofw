package backoff

import (
	"math"
	"math/rand"
	"time"
)

// ExponentialBackoff implements exponential backoff with optional jitter.
// The delay increases exponentially with each retry: initial * (factor ^ retry).
// Jitter adds randomness to prevent thundering herd problem.
type ExponentialBackoff struct {
	// Initial is the starting delay for the first retry
	Initial time.Duration

	// Max is the maximum delay cap (prevents unbounded growth)
	Max time.Duration

	// Factor is the multiplier for each retry (typically 2.0 for doubling)
	// Default: 2.0 if not set
	Factor float64

	// Jitter adds randomness to the delay to prevent thundering herd.
	// When enabled, the actual delay will be randomly selected from [0, calculated_delay].
	Jitter bool
}

// Next calculates the exponential delay for the given retry attempt.
func (e *ExponentialBackoff) Next(retry int) time.Duration {
	// Default factor to 2.0 if not set
	factor := e.Factor
	if factor == 0 {
		factor = 2.0
	}

	// Calculate exponential delay: initial * (factor ^ retry)
	delay := float64(e.Initial) * math.Pow(factor, float64(retry))

	// Cap at maximum
	if e.Max > 0 && time.Duration(delay) > e.Max {
		delay = float64(e.Max)
	}

	// Apply jitter if enabled
	if e.Jitter {
		// Random value between 0 and delay
		delay = rand.Float64() * delay
	}

	return time.Duration(delay)
}

// NewExponentialBackoff creates an exponential backoff with sensible defaults.
// Default configuration:
// - Initial: 100ms
// - Max: 30s
// - Factor: 2.0 (doubling)
// - Jitter: enabled
func NewExponentialBackoff() *ExponentialBackoff {
	return &ExponentialBackoff{
		Initial: 100 * time.Millisecond,
		Max:     30 * time.Second,
		Factor:  2.0,
		Jitter:  true,
	}
}
