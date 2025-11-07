package policy

import (
	"context"
	"errors"
	"net/http"
	"sync"
)

// BulkheadConfig configures the bulkhead (concurrency limiting) behavior.
type BulkheadConfig struct {
	// MaxConcurrent is the maximum number of concurrent requests allowed.
	// Default: 100
	MaxConcurrent int

	// PerHost when true, applies the concurrency limit per target host.
	// When false, applies globally across all hosts.
	// Default: true (per-host isolation)
	PerHost bool
}

// bulkhead represents a single semaphore for concurrency control.
type bulkhead struct {
	semaphore chan struct{}
	maxSize   int
}

// BulkheadPolicy implements concurrency limiting to prevent resource exhaustion.
// It uses a semaphore pattern (buffered channel) to limit concurrent requests.
type BulkheadPolicy struct {
	mu         sync.RWMutex
	bulkheads  map[string]*bulkhead // host -> bulkhead (if PerHost=true)
	global     *bulkhead            // global bulkhead (if PerHost=false)
	config     BulkheadConfig
}

// NewBulkheadPolicy creates a new bulkhead policy with the given configuration.
func NewBulkheadPolicy(config BulkheadConfig) *BulkheadPolicy {
	// Set defaults
	if config.MaxConcurrent == 0 {
		config.MaxConcurrent = 100
	}

	bp := &BulkheadPolicy{
		config: config,
	}

	if config.PerHost {
		bp.bulkheads = make(map[string]*bulkhead)
	} else {
		// Create global bulkhead
		bp.global = newBulkhead(config.MaxConcurrent)
	}

	return bp
}

// Execute implements the Policy interface by limiting concurrency.
func (bp *BulkheadPolicy) Execute(ctx context.Context, req *http.Request, next Executor) (*http.Response, error) {
	// Get the appropriate bulkhead
	var b *bulkhead
	if bp.config.PerHost {
		b = bp.getBulkheadForHost(req.URL.Host)
	} else {
		b = bp.global
	}

	// Try to acquire semaphore (non-blocking)
	select {
	case b.semaphore <- struct{}{}:
		// Acquired - release when done
		defer func() {
			<-b.semaphore
		}()

		// Execute request
		return next(ctx, req)

	default:
		// Semaphore full - fail fast
		return nil, errors.New("bulkhead capacity exceeded")
	}
}

// getBulkheadForHost returns the bulkhead for a given host, creating one if needed.
func (bp *BulkheadPolicy) getBulkheadForHost(host string) *bulkhead {
	bp.mu.RLock()
	b, exists := bp.bulkheads[host]
	bp.mu.RUnlock()

	if exists {
		return b
	}

	// Create new bulkhead
	bp.mu.Lock()
	defer bp.mu.Unlock()

	// Double-check after acquiring write lock
	if b, exists := bp.bulkheads[host]; exists {
		return b
	}

	b = newBulkhead(bp.config.MaxConcurrent)
	bp.bulkheads[host] = b

	return b
}

// newBulkhead creates a new bulkhead with the specified capacity.
func newBulkhead(maxConcurrent int) *bulkhead {
	return &bulkhead{
		semaphore: make(chan struct{}, maxConcurrent),
		maxSize:   maxConcurrent,
	}
}

// ActiveRequests returns the number of currently active requests for a given host.
// Returns 0 if host doesn't exist or if using global bulkhead.
func (bp *BulkheadPolicy) ActiveRequests(host string) int {
	if bp.config.PerHost {
		bp.mu.RLock()
		b, exists := bp.bulkheads[host]
		bp.mu.RUnlock()

		if !exists {
			return 0
		}

		return len(b.semaphore)
	}

	// Global bulkhead
	if bp.global != nil {
		return len(bp.global.semaphore)
	}

	return 0
}
