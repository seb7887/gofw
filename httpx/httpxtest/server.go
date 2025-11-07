package httpxtest

import (
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

// TestServerConfig configures the behavior of a test HTTP server.
type TestServerConfig struct {
	// Latency is the fixed delay before responding
	Latency time.Duration

	// FailureRate is the probability (0.0-1.0) of returning an error response
	FailureRate float64

	// StatusCodes is a list of status codes to rotate through
	// If empty, defaults to [200]
	StatusCodes []int

	// Handler is a custom handler function
	// If set, overrides all other configuration
	Handler http.HandlerFunc
}

// TestServer is a configurable HTTP test server.
type TestServer struct {
	*httptest.Server

	mu            sync.Mutex
	config        TestServerConfig
	requestCount  int
	statusCodeIdx int
}

// NewTestServer creates a new test server with the given configuration.
func NewTestServer(config TestServerConfig) *TestServer {
	// Set defaults
	if len(config.StatusCodes) == 0 {
		config.StatusCodes = []int{http.StatusOK}
	}

	ts := &TestServer{
		config: config,
	}

	// Create httptest server
	ts.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts.handleRequest(w, r)
	}))

	return ts
}

// handleRequest handles an incoming request according to the configuration.
func (ts *TestServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.requestCount++

	// Use custom handler if provided
	if ts.config.Handler != nil {
		ts.config.Handler(w, r)
		return
	}

	// Apply latency
	if ts.config.Latency > 0 {
		time.Sleep(ts.config.Latency)
	}

	// Determine if this request should fail
	shouldFail := rand.Float64() < ts.config.FailureRate

	if shouldFail {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	// Rotate through status codes
	statusCode := ts.config.StatusCodes[ts.statusCodeIdx%len(ts.config.StatusCodes)]
	ts.statusCodeIdx++

	w.WriteHeader(statusCode)
}

// RequestCount returns the total number of requests handled by this server.
func (ts *TestServer) RequestCount() int {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.requestCount
}

// Reset resets the request count.
func (ts *TestServer) Reset() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.requestCount = 0
	ts.statusCodeIdx = 0
}

// TestServerOption is a functional option for configuring a test server.
type TestServerOption func(*TestServerConfig)

// WithLatency sets a fixed latency for all responses.
func WithLatency(d time.Duration) TestServerOption {
	return func(c *TestServerConfig) {
		c.Latency = d
	}
}

// WithFailureRate sets the probability of returning error responses.
// rate should be between 0.0 and 1.0.
func WithFailureRate(rate float64) TestServerOption {
	return func(c *TestServerConfig) {
		c.FailureRate = rate
	}
}

// WithStatusCodes sets the status codes to rotate through.
func WithStatusCodes(codes ...int) TestServerOption {
	return func(c *TestServerConfig) {
		c.StatusCodes = codes
	}
}

// WithHandler sets a custom handler function.
func WithHandler(handler http.HandlerFunc) TestServerOption {
	return func(c *TestServerConfig) {
		c.Handler = handler
	}
}

// NewTestServerWithOptions creates a test server with functional options.
func NewTestServerWithOptions(opts ...TestServerOption) *TestServer {
	config := TestServerConfig{}
	for _, opt := range opts {
		opt(&config)
	}
	return NewTestServer(config)
}
