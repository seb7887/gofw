package policy

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/seb7887/gofw/httpx/backoff"
)

// RetryConfig configures the retry policy behavior.
type RetryConfig struct {
	// MaxAttempts is the maximum number of attempts (including the initial request).
	// Default: 3
	MaxAttempts int

	// Backoff strategy for calculating delay between retries.
	// Default: Exponential backoff with jitter
	Backoff backoff.Backoff

	// ShouldRetry is a custom function to determine if a request should be retried.
	// If nil, uses the default retry condition (network errors + 5xx + 429).
	ShouldRetry func(*http.Response, error) bool

	// RetryableStatusCodes defines which HTTP status codes should be retried.
	// Default: 429 (rate limit), 500-599 (server errors)
	RetryableStatusCodes []int

	// OnlyIdempotent when true, only retries idempotent methods (GET, PUT, DELETE, HEAD, OPTIONS).
	// POST is not retried unless explicitly opted in via request options.
	// Default: true
	OnlyIdempotent bool
}

// RetryPolicy implements automatic retry with configurable backoff strategies.
type RetryPolicy struct {
	config RetryConfig
}

// NewRetryPolicy creates a new retry policy with the given configuration.
func NewRetryPolicy(config RetryConfig) *RetryPolicy {
	// Set defaults
	if config.MaxAttempts == 0 {
		config.MaxAttempts = 3
	}

	if config.Backoff == nil {
		config.Backoff = backoff.NewExponentialBackoff()
	}

	if config.RetryableStatusCodes == nil {
		config.RetryableStatusCodes = []int{429, 500, 502, 503, 504}
	}

	return &RetryPolicy{
		config: config,
	}
}

// Execute implements the Policy interface by retrying failed requests.
func (r *RetryPolicy) Execute(ctx context.Context, req *http.Request, next Executor) (*http.Response, error) {
	var lastResp *http.Response
	var lastErr error

	// Check if method is idempotent
	if r.config.OnlyIdempotent && !isIdempotent(req.Method) {
		// Non-idempotent method - execute once without retry
		return next(ctx, req)
	}

	// Preserve request body for retries
	var bodyBytes []byte
	if req.Body != nil {
		bodyBytes, lastErr = io.ReadAll(req.Body)
		if lastErr != nil {
			return nil, lastErr
		}
		req.Body.Close()
	}

	// Attempt the request up to MaxAttempts times
	for attempt := 0; attempt < r.config.MaxAttempts; attempt++ {
		// Restore body for each attempt
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		// Execute request
		lastResp, lastErr = next(ctx, req)

		// Check if we should retry
		shouldRetry := r.shouldRetry(lastResp, lastErr)

		// Success or non-retriable error - return immediately
		if !shouldRetry {
			return lastResp, lastErr
		}

		// Close response body if present to avoid resource leak
		if lastResp != nil && lastResp.Body != nil {
			io.Copy(io.Discard, lastResp.Body)
			lastResp.Body.Close()
		}

		// Don't sleep after the last attempt
		if attempt < r.config.MaxAttempts-1 {
			// Calculate backoff delay
			delay := r.config.Backoff.Next(attempt)

			// Wait for backoff period or context cancellation
			select {
			case <-time.After(delay):
				// Continue to next attempt
			case <-ctx.Done():
				// Context cancelled - return context error
				return nil, ctx.Err()
			}
		}
	}

	// All retries exhausted
	return lastResp, errors.Join(lastErr, errors.New("max retry attempts exceeded"))
}

// shouldRetry determines if a request should be retried based on response and error.
func (r *RetryPolicy) shouldRetry(resp *http.Response, err error) bool {
	// Use custom retry condition if provided
	if r.config.ShouldRetry != nil {
		return r.config.ShouldRetry(resp, err)
	}

	// Network error - always retry
	if err != nil {
		return true
	}

	// Check status code
	if resp != nil {
		for _, code := range r.config.RetryableStatusCodes {
			if resp.StatusCode == code {
				return true
			}
		}
	}

	// Success or non-retriable error
	return false
}

// isIdempotent returns true if the HTTP method is idempotent.
// Idempotent methods: GET, PUT, DELETE, HEAD, OPTIONS, TRACE
// Non-idempotent: POST, PATCH
func isIdempotent(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodDelete,
		http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}
