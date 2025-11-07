package httpxtest

import (
	"context"
	"net/http"
	"sync"
)

// MockTransport is a mock implementation of httpx.Transport for testing.
// It allows configuring response behavior and capturing request history.
type MockTransport struct {
	mu sync.Mutex

	// Response to return (if Err is nil)
	Response *http.Response

	// Err to return (takes precedence over Response)
	Err error

	// Func is a custom function to handle requests
	// If set, takes precedence over Response and Err
	Func func(ctx context.Context, req *http.Request) (*http.Response, error)

	// Requests captures all requests made to this transport
	Requests []*http.Request

	// CallCount tracks the number of times Do() was called
	CallCount int
}

// Do implements the Transport interface.
func (m *MockTransport) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CallCount++
	m.Requests = append(m.Requests, req)

	// Use custom function if provided
	if m.Func != nil {
		return m.Func(ctx, req)
	}

	// Return error if set
	if m.Err != nil {
		return nil, m.Err
	}

	// Return configured response
	return m.Response, nil
}

// Reset clears the request history and call count.
func (m *MockTransport) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Requests = nil
	m.CallCount = 0
}

// LastRequest returns the most recent request, or nil if no requests have been made.
func (m *MockTransport) LastRequest() *http.Request {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.Requests) == 0 {
		return nil
	}

	return m.Requests[len(m.Requests)-1]
}
