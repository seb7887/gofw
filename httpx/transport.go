package httpx

import (
	"context"
	"net/http"
	"time"
)

// Transport abstracts the actual HTTP request execution.
// This interface allows for easy mocking and testing of HTTP interactions.
type Transport interface {
	// Do executes an HTTP request and returns the response.
	// The context can be used for cancellation and timeout control.
	Do(ctx context.Context, req *http.Request) (*http.Response, error)
}

// DefaultTransport wraps the standard library's http.Client.
type DefaultTransport struct {
	client *http.Client
}

// NewDefaultTransport creates a new transport with sensible defaults.
// The underlying http.Client is configured with:
// - Connection pooling (100 max idle connections, 10 per host)
// - 90 second idle connection timeout
// - HTTP/2 enabled by default
func NewDefaultTransport() *DefaultTransport {
	return &DefaultTransport{
		client: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				// HTTP/2 is enabled by default in Go 1.6+
			},
		},
	}
}

// NewDefaultTransportWithClient creates a transport using a custom http.Client.
// This allows full control over connection pooling, TLS configuration, proxies, etc.
func NewDefaultTransportWithClient(client *http.Client) *DefaultTransport {
	return &DefaultTransport{
		client: client,
	}
}

// Do implements the Transport interface by delegating to the underlying http.Client.
func (t *DefaultTransport) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Clone the request with the provided context
	req = req.WithContext(ctx)
	return t.client.Do(req)
}
