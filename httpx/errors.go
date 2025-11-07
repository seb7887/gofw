package httpx

import (
	"errors"
	"fmt"
	"net/http"
)

// Sentinel errors that can be checked using errors.Is
var (
	// ErrCircuitOpen is returned when a circuit breaker is in the open state.
	ErrCircuitOpen = errors.New("circuit breaker is open")

	// ErrBulkheadFull is returned when the bulkhead capacity is exceeded.
	ErrBulkheadFull = errors.New("bulkhead capacity exceeded")

	// ErrTimeout is returned when a request times out.
	ErrTimeout = errors.New("request timeout")

	// ErrMaxRetriesExceeded is returned when all retry attempts have been exhausted.
	ErrMaxRetriesExceeded = errors.New("max retry attempts exceeded")
)

// RequestError provides rich context about failed HTTP requests.
// It wraps the underlying error with additional information about the request,
// response, retry attempts, and error category.
type RequestError struct {
	// Err is the underlying error that caused the request to fail
	Err error

	// Request is the original HTTP request (may be nil if error occurred before request creation)
	Request *http.Request

	// Response is the HTTP response if one was received (may be nil for network errors)
	Response *http.Response

	// Retries is the number of retry attempts that were made
	Retries int

	// Cause categorizes the error for easier handling.
	// Common values: "circuit_open", "timeout", "network", "max_retries", "bulkhead_full"
	Cause string
}

// Error implements the error interface.
func (e *RequestError) Error() string {
	if e.Request != nil {
		return fmt.Sprintf("httpx: %s %s failed: %s (cause: %s, retries: %d)",
			e.Request.Method,
			e.Request.URL.String(),
			e.Err.Error(),
			e.Cause,
			e.Retries,
		)
	}
	return fmt.Sprintf("httpx: request failed: %s (cause: %s, retries: %d)",
		e.Err.Error(),
		e.Cause,
		e.Retries,
	)
}

// Unwrap returns the underlying error, allowing errors.Is and errors.As to work.
func (e *RequestError) Unwrap() error {
	return e.Err
}

// StatusCode returns the HTTP status code from the response if available.
// Returns 0 if no response was received.
func (e *RequestError) StatusCode() int {
	if e.Response != nil {
		return e.Response.StatusCode
	}
	return 0
}
