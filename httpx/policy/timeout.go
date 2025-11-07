package policy

import (
	"context"
	"errors"
	"net/http"
	"time"
)

// TimeoutConfig configures timeout behavior at multiple levels.
type TimeoutConfig struct {
	// Request is the total timeout for the entire request (including retries).
	// If 0, no timeout is applied at the policy level.
	// Default: 30 seconds
	Request time.Duration

	// Connection timeout is handled at the transport level (http.Transport.DialContext)
	// TLS handshake timeout is also handled at transport level
	// These are configured via WithHTTPClient option, not in this policy
}

// TimeoutPolicy implements timeout controls for HTTP requests.
// It wraps the request execution with a context deadline.
type TimeoutPolicy struct {
	config TimeoutConfig
}

// NewTimeoutPolicy creates a new timeout policy with the given configuration.
func NewTimeoutPolicy(config TimeoutConfig) *TimeoutPolicy {
	// Set defaults
	if config.Request == 0 {
		config.Request = 30 * time.Second
	}

	return &TimeoutPolicy{
		config: config,
	}
}

// Execute implements the Policy interface by applying timeout to the request.
func (t *TimeoutPolicy) Execute(ctx context.Context, req *http.Request, next Executor) (*http.Response, error) {
	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, t.config.Request)
	defer cancel()

	// Execute request with timeout context
	resp, err := next(timeoutCtx, req)

	// Check if timeout occurred
	if err != nil && errors.Is(err, context.DeadlineExceeded) {
		return nil, errors.New("request timeout")
	}

	return resp, err
}
