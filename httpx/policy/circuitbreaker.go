package policy

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"
)

// CircuitState represents the state of a circuit breaker.
type CircuitState int

const (
	// StateClosed: Circuit is closed, requests pass through normally
	StateClosed CircuitState = iota

	// StateOpen: Circuit is open, requests fail-fast without hitting downstream
	StateOpen

	// StateHalfOpen: Circuit is testing if service recovered, limited requests allowed
	StateHalfOpen
)

// String returns the string representation of the circuit state.
func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig configures the circuit breaker behavior.
type CircuitBreakerConfig struct {
	// ErrorThreshold is the percentage of errors (0-100) that triggers the circuit to open.
	// Default: 50
	ErrorThreshold int

	// MinRequests is the minimum number of requests before evaluating error threshold.
	// This prevents opening the circuit on low traffic.
	// Default: 10
	MinRequests int

	// SleepWindow is the time to wait in open state before transitioning to half-open.
	// Default: 5 seconds
	SleepWindow time.Duration

	// SuccessThreshold is the number of consecutive successes in half-open state
	// required to close the circuit.
	// Default: 2
	SuccessThreshold int

	// ShouldTrip is a custom function to determine if an error should count toward opening the circuit.
	// If nil, all errors and 5xx status codes count as failures.
	ShouldTrip func(*http.Response, error) bool
}

// circuitBreaker maintains the state for a single circuit.
type circuitBreaker struct {
	mu sync.RWMutex

	state            CircuitState
	failures         int
	successes        int
	requests         int
	lastStateChange  time.Time
	config           CircuitBreakerConfig
}

// CircuitBreakerPolicy implements the circuit breaker pattern to prevent cascading failures.
// It maintains per-host circuit breakers to isolate failures by service.
type CircuitBreakerPolicy struct {
	mu       sync.RWMutex
	breakers map[string]*circuitBreaker // host -> circuit breaker
	config   CircuitBreakerConfig
}

// NewCircuitBreakerPolicy creates a new circuit breaker policy with the given configuration.
func NewCircuitBreakerPolicy(config CircuitBreakerConfig) *CircuitBreakerPolicy {
	// Set defaults
	if config.ErrorThreshold == 0 {
		config.ErrorThreshold = 50
	}
	if config.MinRequests == 0 {
		config.MinRequests = 10
	}
	if config.SleepWindow == 0 {
		config.SleepWindow = 5 * time.Second
	}
	if config.SuccessThreshold == 0 {
		config.SuccessThreshold = 2
	}

	return &CircuitBreakerPolicy{
		breakers: make(map[string]*circuitBreaker),
		config:   config,
	}
}

// Execute implements the Policy interface by checking circuit breaker state.
func (cb *CircuitBreakerPolicy) Execute(ctx context.Context, req *http.Request, next Executor) (*http.Response, error) {
	// Get or create circuit breaker for this host
	breaker := cb.getBreakerForHost(req.URL.Host)

	// Check if circuit is open
	if !breaker.canExecute() {
		return nil, errors.New("circuit breaker is open")
	}

	// Execute request
	resp, err := next(ctx, req)

	// Record result
	shouldTrip := cb.shouldTrip(resp, err)
	breaker.recordResult(shouldTrip)

	return resp, err
}

// getBreakerForHost returns the circuit breaker for a given host, creating one if needed.
func (cb *CircuitBreakerPolicy) getBreakerForHost(host string) *circuitBreaker {
	cb.mu.RLock()
	breaker, exists := cb.breakers[host]
	cb.mu.RUnlock()

	if exists {
		return breaker
	}

	// Create new breaker
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Double-check after acquiring write lock
	if breaker, exists := cb.breakers[host]; exists {
		return breaker
	}

	breaker = &circuitBreaker{
		state:           StateClosed,
		config:          cb.config,
		lastStateChange: time.Now(),
	}
	cb.breakers[host] = breaker

	return breaker
}

// shouldTrip determines if a response/error should count as a failure.
func (cb *CircuitBreakerPolicy) shouldTrip(resp *http.Response, err error) bool {
	// Use custom trip condition if provided
	if cb.config.ShouldTrip != nil {
		return cb.config.ShouldTrip(resp, err)
	}

	// Network error - always counts as failure
	if err != nil {
		return true
	}

	// 5xx status codes count as failures
	if resp != nil && resp.StatusCode >= 500 {
		return true
	}

	// Success
	return false
}

// canExecute checks if the circuit breaker allows execution.
func (b *circuitBreaker) canExecute() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateClosed:
		// Always allow in closed state
		return true

	case StateOpen:
		// Check if sleep window has passed
		if time.Since(b.lastStateChange) > b.config.SleepWindow {
			// Transition to half-open
			b.state = StateHalfOpen
			b.successes = 0
			b.failures = 0
			b.requests = 0
			b.lastStateChange = time.Now()
			return true
		}
		// Still in sleep window - fail fast
		return false

	case StateHalfOpen:
		// Allow request in half-open state (to test if service recovered)
		return true

	default:
		return false
	}
}

// recordResult records the result of a request and updates circuit state.
func (b *circuitBreaker) recordResult(isFailure bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.requests++

	if isFailure {
		b.failures++
		b.handleFailure()
	} else {
		b.successes++
		b.handleSuccess()
	}
}

// handleFailure handles a failed request based on current state.
func (b *circuitBreaker) handleFailure() {
	switch b.state {
	case StateClosed:
		// Check if we should open the circuit
		if b.requests >= b.config.MinRequests {
			errorRate := (b.failures * 100) / b.requests
			if errorRate >= b.config.ErrorThreshold {
				// Open the circuit
				b.state = StateOpen
				b.lastStateChange = time.Now()
			}
		}

	case StateHalfOpen:
		// Any failure in half-open state reopens the circuit
		b.state = StateOpen
		b.successes = 0
		b.failures = 0
		b.requests = 0
		b.lastStateChange = time.Now()
	}
}

// handleSuccess handles a successful request based on current state.
func (b *circuitBreaker) handleSuccess() {
	switch b.state {
	case StateHalfOpen:
		// Check if we have enough successes to close the circuit
		if b.successes >= b.config.SuccessThreshold {
			// Close the circuit
			b.state = StateClosed
			b.successes = 0
			b.failures = 0
			b.requests = 0
			b.lastStateChange = time.Now()
		}
	}
}

// State returns the current state of the circuit breaker for a given host.
// This is useful for metrics and monitoring.
func (cb *CircuitBreakerPolicy) State(host string) CircuitState {
	cb.mu.RLock()
	breaker, exists := cb.breakers[host]
	cb.mu.RUnlock()

	if !exists {
		return StateClosed
	}

	breaker.mu.RLock()
	defer breaker.mu.RUnlock()
	return breaker.state
}
