package httpx

import (
	"math"
	"math/rand"
	"time"
)

type Backoff interface {
	Next(retry int) time.Duration
}

type constantBackoff struct {
	backoffInterval   int64
	maxJitterInterval int64
}

func init() {
	rand.NewSource(time.Now().UnixNano())
}

func NewConstantBackoff(backoffInterval, maxJitterInterval int64) Backoff {
	if maxJitterInterval < 0 {
		maxJitterInterval = 0
	}

	return &constantBackoff{
		backoffInterval:   backoffInterval / int64(time.Millisecond),
		maxJitterInterval: maxJitterInterval / int64(time.Millisecond),
	}
}

func (b *constantBackoff) Next(_ int) time.Duration {
	return (time.Duration(b.backoffInterval) * time.Millisecond) + (time.Duration(rand.Int63n(b.maxJitterInterval+1)) * time.Millisecond)
}

type exponentialBackoff struct {
	exponentFactor float64
	initialDelay   float64
	maxDelay       float64
	maxJitter      int64
}

func NewExponentialBackoff(initialTimeout, maxTimeout, maxJitter time.Duration, exponentFactor float64) Backoff {
	if maxJitter < 0 {
		maxJitter = 0
	}

	return &exponentialBackoff{
		exponentFactor: exponentFactor,
		initialDelay:   float64(initialTimeout / time.Millisecond),
		maxDelay:       float64(maxTimeout / time.Millisecond),
		maxJitter:      int64(maxJitter / time.Millisecond),
	}
}

func (b *exponentialBackoff) Next(retry int) time.Duration {
	if retry < 0 {
		retry = 0
	}
	return time.Duration(math.Min(b.initialDelay*math.Pow(b.exponentFactor, float64(retry)), b.maxDelay)+float64(rand.Int63n(b.maxJitter+1))) * time.Millisecond
}
