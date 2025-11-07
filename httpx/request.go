package httpx

import (
	"io"
	"net/http"
	"time"
)

// Headers is a convenience type for HTTP headers.
type Headers map[string]string

// Request represents an HTTP request with additional configuration options.
type Request struct {
	// Method is the HTTP method (GET, POST, PUT, PATCH, DELETE, etc.)
	Method string

	// Path is the URL path (will be joined with Client's BaseURL if set)
	Path string

	// Headers are the HTTP headers to send with the request
	Headers Headers

	// Body is the request body (for POST, PUT, PATCH requests)
	Body io.Reader

	// Options allow per-request overrides of client-level policies
	Options []RequestOption
}

// RequestOption allows per-request configuration overrides.
// These options can override the client's default behavior for specific requests.
type RequestOption interface {
	apply(*requestConfig)
}

// requestConfig holds per-request configuration that can override client defaults.
type requestConfig struct {
	// Timeout overrides the client's default request timeout
	timeout *time.Duration

	// Retryable explicitly enables/disables retry for this request
	retryable *bool

	// DisableCircuitBreaker disables circuit breaker for this request
	disableCircuitBreaker bool

	// DisableRetry disables retry policy for this request
	disableRetry bool

	// DisableTimeout disables timeout policy for this request
	disableTimeout bool

	// DisableBulkhead disables bulkhead policy for this request
	disableBulkhead bool
}

// funcOption wraps a function to implement RequestOption
type funcOption struct {
	f func(*requestConfig)
}

func (fo *funcOption) apply(cfg *requestConfig) {
	fo.f(cfg)
}

// WithTimeout overrides the client's default timeout for this specific request.
func WithTimeout(d time.Duration) RequestOption {
	return &funcOption{
		f: func(cfg *requestConfig) {
			cfg.timeout = &d
		},
	}
}

// WithRetryable explicitly enables or disables retry for this request.
// This is useful for:
// - Enabling retry on POST requests (which are non-idempotent by default)
// - Disabling retry on specific requests even if the client has retry enabled
func WithRetryable(retryable bool) RequestOption {
	return &funcOption{
		f: func(cfg *requestConfig) {
			cfg.retryable = &retryable
		},
	}
}

// WithoutCircuitBreaker disables the circuit breaker policy for this request.
// Use this when you want to bypass the circuit breaker for critical requests
// (e.g., health checks, authentication).
func WithoutCircuitBreaker() RequestOption {
	return &funcOption{
		f: func(cfg *requestConfig) {
			cfg.disableCircuitBreaker = true
		},
	}
}

// WithoutRetry disables the retry policy for this request.
func WithoutRetry() RequestOption {
	return &funcOption{
		f: func(cfg *requestConfig) {
			cfg.disableRetry = true
		},
	}
}

// WithoutTimeout disables the timeout policy for this request.
// Use with caution - only for requests that truly have no time bound.
func WithoutTimeout() RequestOption {
	return &funcOption{
		f: func(cfg *requestConfig) {
			cfg.disableTimeout = true
		},
	}
}

// WithoutBulkhead disables the bulkhead policy for this request.
// Use for critical requests that should bypass concurrency limits.
func WithoutBulkhead() RequestOption {
	return &funcOption{
		f: func(cfg *requestConfig) {
			cfg.disableBulkhead = true
		},
	}
}

// applyOptions applies all request options to the config.
func applyOptions(opts []RequestOption) *requestConfig {
	cfg := &requestConfig{}
	for _, opt := range opts {
		opt.apply(cfg)
	}
	return cfg
}

// toHTTPRequest converts a Request to a standard http.Request.
func (r *Request) toHTTPRequest(baseURL string) (*http.Request, error) {
	// Build full URL
	url := baseURL + r.Path

	// Create HTTP request
	req, err := http.NewRequest(r.Method, url, r.Body)
	if err != nil {
		return nil, err
	}

	// Add headers
	for key, value := range r.Headers {
		req.Header.Set(key, value)
	}

	return req, nil
}
