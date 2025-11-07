package httpx

import (
	"context"
	"io"
	"net/http"

	"github.com/seb7887/gofw/httpx/policy"
)

// Client is the main HTTP client that orchestrates transport and policies.
// It is thread-safe and immutable after creation.
type Client struct {
	// transport is the underlying HTTP executor
	transport Transport

	// baseURL is prepended to all request paths
	baseURL string

	// policies is the chain of resilience policies
	policies []policy.Policy

	// executor is the final chained executor (policies + transport)
	executor policy.Executor
}

// NewClient creates a new HTTP client with the provided options.
// The client is configured using functional options pattern.
//
// Example:
//
//	client := httpx.NewClient(
//	    httpx.WithBaseURL("http://service-b:8080"),
//	    httpx.WithRetry(httpx.RetryConfig{MaxAttempts: 3}),
//	    httpx.WithCircuitBreaker(httpx.CircuitBreakerConfig{...}),
//	)
func NewClient(opts ...ClientOption) *Client {
	// Default configuration
	c := &Client{
		transport: NewDefaultTransport(),
		baseURL:   "",
		policies:  []policy.Policy{},
	}

	// Apply options
	for _, opt := range opts {
		opt.apply(c)
	}

	// Build the policy chain
	c.executor = policy.Chain(c.policies, c.transport.Do)

	return c
}

// Do executes an HTTP request with all configured policies applied.
// This is the most flexible method, allowing full control over the request.
func (c *Client) Do(ctx context.Context, req *Request) (*http.Response, error) {
	// Convert to http.Request
	httpReq, err := req.toHTTPRequest(c.baseURL)
	if err != nil {
		return nil, &RequestError{
			Err:     err,
			Request: nil,
			Cause:   "invalid_request",
		}
	}

	// TODO: Apply per-request options (timeout overrides, policy disabling, etc)
	// For now, just execute with the client's policy chain

	return c.executor(ctx, httpReq)
}

// Get executes a GET request to the specified path.
// Headers are optional and can be nil.
func (c *Client) Get(ctx context.Context, path string, headers ...Headers) (*http.Response, error) {
	h := Headers{}
	if len(headers) > 0 {
		h = headers[0]
	}

	return c.Do(ctx, &Request{
		Method:  http.MethodGet,
		Path:    path,
		Headers: h,
	})
}

// Post executes a POST request to the specified path with the given body.
// Headers are optional and can be nil.
func (c *Client) Post(ctx context.Context, path string, headers Headers, body io.Reader) (*http.Response, error) {
	if headers == nil {
		headers = Headers{}
	}

	return c.Do(ctx, &Request{
		Method:  http.MethodPost,
		Path:    path,
		Headers: headers,
		Body:    body,
	})
}

// Put executes a PUT request to the specified path with the given body.
// Headers are optional and can be nil.
func (c *Client) Put(ctx context.Context, path string, headers Headers, body io.Reader) (*http.Response, error) {
	if headers == nil {
		headers = Headers{}
	}

	return c.Do(ctx, &Request{
		Method:  http.MethodPut,
		Path:    path,
		Headers: headers,
		Body:    body,
	})
}

// Patch executes a PATCH request to the specified path with the given body.
// Headers are optional and can be nil.
func (c *Client) Patch(ctx context.Context, path string, headers Headers, body io.Reader) (*http.Response, error) {
	if headers == nil {
		headers = Headers{}
	}

	return c.Do(ctx, &Request{
		Method:  http.MethodPatch,
		Path:    path,
		Headers: headers,
		Body:    body,
	})
}

// Delete executes a DELETE request to the specified path.
// Headers are optional and can be nil.
func (c *Client) Delete(ctx context.Context, path string, headers ...Headers) (*http.Response, error) {
	h := Headers{}
	if len(headers) > 0 {
		h = headers[0]
	}

	return c.Do(ctx, &Request{
		Method:  http.MethodDelete,
		Path:    path,
		Headers: h,
	})
}
