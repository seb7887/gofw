# httpx: Resilient HTTP Client Design

**Date**: 2025-11-05
**Status**: Approved
**Author**: Design Session with Claude Code

## Executive Summary

The **httpx** package is a resilient HTTP client library for Go microservices that implements production-grade reliability patterns including circuit breakers, intelligent retry logic, granular timeouts, and bulkhead isolation. It provides deep observability through OpenTelemetry distributed tracing and Prometheus metrics.

### Key Objectives

- **Prevent cascading failures** in microservice architectures through circuit breaker pattern
- **Intelligent retry handling** with configurable backoff strategies
- **Fine-grained timeout control** to prevent hanging requests
- **Concurrency limiting** via bulkhead pattern for resource protection
- **Deep observability** with OpenTelemetry traces and Prometheus metrics
- **Clean, idiomatic Go API** using builder pattern and functional options

## Architecture

### High-Level Design: Policy-Based Decorator Pattern

The architecture separates HTTP execution from resilience policies through a three-layer design:

```
┌─────────────────────────────────────────────────────────┐
│                      Client API                          │
│  (Builder, convenience methods, OTEL integration)        │
└─────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│                    Policy Chain                          │
│  CircuitBreaker → Retry → Timeout → Bulkhead            │
│  (Each policy wraps the next via Decorator pattern)     │
└─────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│                  Transport Layer                         │
│  (Wraps net/http.Client, handles HTTP execution)        │
└─────────────────────────────────────────────────────────┘
```

### Core Interfaces

#### Transport Interface
```go
type Transport interface {
    Do(ctx context.Context, req *http.Request) (*http.Response, error)
}
```

Abstracts the actual HTTP execution. Default implementation wraps `*http.Client` from stdlib.

#### Policy Interface
```go
type Policy interface {
    Execute(ctx context.Context, req *http.Request, next Executor) (*http.Response, error)
}

type Executor func(ctx context.Context, req *http.Request) (*http.Response, error)
```

Policies are chained decorators that can:
- Execute the next policy/transport in the chain
- Short-circuit and return early (e.g., circuit breaker open)
- Retry by calling `next()` multiple times
- Modify requests/responses
- Record metrics and traces

## Resilience Policies

### 1. Circuit Breaker Policy

Implements a state machine to prevent cascading failures when downstream services are unhealthy.

**States**:
- **Closed**: Normal operation, requests pass through
- **Open**: Service unhealthy, requests fail-fast without hitting downstream
- **Half-Open**: Testing if service recovered, limited requests allowed

**Configuration**:
```go
type CircuitBreakerConfig struct {
    ErrorThreshold      int           // % of errors to trigger open state (default: 50)
    MinRequests         int           // Min requests before evaluating (default: 10)
    SleepWindow         time.Duration // Time in open before half-open (default: 5s)
    SuccessThreshold    int           // Successes in half-open to close (default: 2)
}
```

**Implementation Details**:
- **Per-host circuit breakers**: Each target host has independent circuit breaker state
- **Thread-safe**: Uses `sync.RWMutex` for concurrent access
- **Metrics exposed**: Current state, failure counts, state transitions

**State Transitions**:
```
Closed ──(errors > threshold)──> Open
Open ──(sleep window expired)──> Half-Open
Half-Open ──(success >= threshold)──> Closed
Half-Open ──(any failure)──> Open
```

### 2. Retry Policy

Automatically retries failed requests with configurable backoff strategies.

**Configuration**:
```go
type RetryConfig struct {
    MaxAttempts    int
    Backoff        Backoff      // Exponential, Linear, or Fixed
    RetryCondition func(*http.Response, error) bool // Custom retry logic
}
```

**Default Retry Conditions**:
- Network errors (connection refused, timeout, DNS failure)
- HTTP 5xx status codes (server errors)
- HTTP 429 (rate limit, with respect to Retry-After header)
- **Idempotent methods only** (GET, PUT, DELETE) by default
- POST requires explicit opt-in via `WithRetryable(true)`

**Backoff Strategies**:

1. **Exponential with Jitter**:
   ```go
   type ExponentialBackoff struct {
       Initial time.Duration  // Starting delay (e.g., 100ms)
       Max     time.Duration  // Maximum delay cap (e.g., 2s)
       Factor  float64        // Multiplier per retry (default: 2.0)
       Jitter  bool           // Add randomness to prevent thundering herd
   }
   ```

2. **Linear**:
   ```go
   type LinearBackoff struct {
       Interval time.Duration // Fixed increment per retry
   }
   ```

**Body Preservation**:
- Request bodies are automatically buffered as `io.ReadSeeker`
- Allows replaying body on retry attempts
- Large bodies (>10MB) trigger warning and may not be retried

### 3. Timeout Policy

Provides granular timeout control at multiple levels.

**Configuration**:
```go
type TimeoutConfig struct {
    Connection       time.Duration // TCP connection timeout
    TLSHandshake     time.Duration // TLS handshake timeout
    ResponseHeader   time.Duration // Time to receive response headers
    Request          time.Duration // Total request timeout (includes retries)
}
```

**Behavior**:
- Uses `context.WithTimeout` for request-level timeouts
- Connection/TLS timeouts configured on `http.Transport`
- Per-request timeout overrides via `RequestOption`
- Timeout expiration propagates cancellation through context

### 4. Bulkhead Policy

Limits concurrent requests to prevent resource exhaustion and isolate failures.

**Configuration**:
```go
type BulkheadConfig struct {
    MaxConcurrent int  // Max concurrent requests (default: 100)
}
```

**Implementation**:
- Semaphore-based using buffered channel
- **Fail-fast behavior**: Returns error immediately if capacity exceeded
- **Per-host isolation**: Each target service has independent semaphore
- **Metrics**: Tracks active requests, queue depth, rejections

**Error Handling**:
- Returns `ErrBulkheadFull` when capacity exceeded
- Does NOT queue requests (fail-fast for predictable latency)

## Observability

### OpenTelemetry Integration

**Distributed Tracing**:
- Automatically creates spans for each HTTP request
- Span name: `HTTP {method}` (e.g., "HTTP GET")
- Span attributes:
  - `http.method`: Request method
  - `http.url`: Full URL
  - `http.status_code`: Response status
  - `peer.service`: Target service name (from host)
  - `http.retry_count`: Number of retry attempts
  - `http.circuit_breaker_state`: Current CB state

**Context Propagation**:
- Injects W3C Trace Context headers automatically
- Propagates trace ID, span ID across service boundaries
- Creates sub-spans for policy execution (retry attempts, CB checks)

**Error Recording**:
- Errors recorded as span events with stack traces
- Span status set to `Error` on failures

**Usage**:
```go
client := httpx.NewClient(
    httpx.WithOTEL(otelProvider),
)

// Trace context automatically propagated
resp, err := client.Get(ctx, "/api/users")
```

### Prometheus Metrics

**Metrics Exposed**:

1. **Request Duration** (Histogram):
   ```
   http_client_request_duration_seconds{method, status_code, host, path_pattern}
   ```
   - Buckets: 0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10 seconds

2. **Circuit Breaker State** (Gauge):
   ```
   http_client_circuit_breaker_state{host}
   ```
   - Values: 0 (closed), 1 (open), 2 (half-open)

3. **Circuit Breaker Failures** (Counter):
   ```
   http_client_circuit_breaker_failures_total{host}
   ```

4. **Retry Attempts** (Counter):
   ```
   http_client_retries_total{method, host, reason}
   ```
   - Reasons: network_error, 5xx, 429, custom

5. **Active Requests** (Gauge):
   ```
   http_client_active_requests{host}
   ```

6. **Bulkhead Rejections** (Counter):
   ```
   http_client_rejected_requests_total{host}
   ```

**Cardinality Control**:
- Path parameters abstracted: `/users/123` → `/users/:id`
- Host normalized (strip port if default)
- Limited status code groups: 2xx, 3xx, 4xx, 5xx

## API Design

### Client Construction (Builder Pattern)

```go
client := httpx.NewClient(
    // Transport configuration
    httpx.WithHTTPClient(&http.Client{
        Transport: &http.Transport{
            MaxIdleConns:        100,
            MaxIdleConnsPerHost: 10,
            IdleConnTimeout:     90 * time.Second,
        },
    }),
    httpx.WithBaseURL("http://service-b:8080"),

    // Resilience policies
    httpx.WithCircuitBreaker(httpx.CircuitBreakerConfig{
        ErrorThreshold:   50,
        MinRequests:      10,
        SleepWindow:      5 * time.Second,
        SuccessThreshold: 2,
    }),
    httpx.WithRetry(httpx.RetryConfig{
        MaxAttempts: 3,
        Backoff: httpx.ExponentialBackoff{
            Initial: 100 * time.Millisecond,
            Max:     2 * time.Second,
            Jitter:  true,
        },
    }),
    httpx.WithTimeout(httpx.TimeoutConfig{
        Connection: 2 * time.Second,
        Request:    10 * time.Second,
    }),
    httpx.WithBulkhead(10), // Max 10 concurrent requests

    // Observability
    httpx.WithOTEL(otelProvider),
    httpx.WithMetrics(promRegistry),
)
```

### Request API

**Convenience Methods**:
```go
// Simple GET
resp, err := client.Get(ctx, "/users/123")

// GET with headers
resp, err := client.Get(ctx, "/users/123",
    httpx.Headers{"Authorization": "Bearer token"},
)

// POST with body
resp, err := client.Post(ctx, "/users",
    httpx.Headers{"Content-Type": "application/json"},
    bytes.NewReader(jsonBody),
)

// Other methods: Put, Patch, Delete
```

**Advanced Request with Options**:
```go
resp, err := client.Do(ctx, &httpx.Request{
    Method:  "POST",
    Path:    "/orders",
    Headers: httpx.Headers{"Content-Type": "application/json"},
    Body:    orderPayload,
    Options: []httpx.RequestOption{
        httpx.WithRetryable(true),          // Override policy default
        httpx.WithTimeout(5 * time.Second), // Override client default
        httpx.WithoutCircuitBreaker(),      // Disable CB for this request
    },
})
```

### Configuration Levels

1. **Global (Client-level)**: Policies configured at client creation apply to all requests
2. **Per-request**: `RequestOption` can override specific behaviors
3. **Immutable**: Client configuration is immutable after creation (thread-safe)

## Error Handling

### Error Types

```go
// Sentinel errors
var (
    ErrCircuitOpen         = errors.New("circuit breaker is open")
    ErrBulkheadFull        = errors.New("bulkhead capacity exceeded")
    ErrTimeout             = errors.New("request timeout")
    ErrMaxRetriesExceeded  = errors.New("max retry attempts exceeded")
)

// Rich error context
type RequestError struct {
    Err       error              // Underlying error
    Request   *http.Request      // Original request
    Response  *http.Response     // Response (may be nil)
    Retries   int                // Number of retry attempts made
    Cause     string             // Error category: "circuit_open", "timeout", etc
}

func (e *RequestError) Error() string
func (e *RequestError) Unwrap() error
```

### Error Handling Patterns

```go
resp, err := client.Get(ctx, "/api/data")
if err != nil {
    var reqErr *httpx.RequestError
    if errors.As(err, &reqErr) {
        switch reqErr.Cause {
        case "circuit_open":
            // Circuit breaker is open, use fallback
        case "timeout":
            // Request timed out
        case "max_retries":
            // Exhausted all retry attempts
        }
    }
}
```

## Testing Strategy

### Unit Tests

**Per-Policy Testing**:
- Mock the `Transport` interface
- Test state transitions (e.g., CB state machine)
- Test edge cases (body buffering, timeout cancellation)
- Test configuration validation

**Example**:
```go
func TestCircuitBreakerStateTransitions(t *testing.T) {
    mockTransport := &MockTransport{
        Err: errors.New("service unavailable"),
    }

    cb := NewCircuitBreakerPolicy(CircuitBreakerConfig{
        ErrorThreshold: 50,
        MinRequests:    5,
    })

    // Make 5 requests that fail
    for i := 0; i < 5; i++ {
        _, err := cb.Execute(ctx, req, mockTransport.Do)
        require.Error(t, err)
    }

    // Circuit should now be open
    assert.Equal(t, StateOpen, cb.State())

    // Next request should fail-fast
    _, err := cb.Execute(ctx, req, mockTransport.Do)
    assert.ErrorIs(t, err, ErrCircuitOpen)
}
```

### Integration Tests

**httptest.Server** for simulating services:
```go
func TestRetryWithRealServer(t *testing.T) {
    attempts := 0
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        attempts++
        if attempts < 3 {
            w.WriteHeader(http.StatusServiceUnavailable)
            return
        }
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()

    client := httpx.NewClient(
        httpx.WithRetry(httpx.RetryConfig{MaxAttempts: 3}),
    )

    resp, err := client.Get(context.Background(), server.URL)
    require.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)
    assert.Equal(t, 3, attempts)
}
```

### Test Utilities (httpxtest subpackage)

```go
// Configurable test server
server := httpxtest.NewTestServer(
    httpxtest.WithLatency(100 * time.Millisecond),
    httpxtest.WithFailureRate(0.3),  // 30% of requests fail
    httpxtest.WithStatusCodes(http.StatusOK, http.StatusServiceUnavailable),
)

// Metric assertions
httpxtest.AssertMetric(t, registry, "http_client_retries_total", 3)
httpxtest.AssertMetricWithLabels(t, registry,
    "http_client_circuit_breaker_state",
    prometheus.Labels{"host": "service-b"},
    1.0, // Open state
)

// OTEL trace assertions
httpxtest.AssertSpanAttribute(t, span, "http.retry_count", 2)
```

### Benchmarks

```go
// Measure overhead of policies
BenchmarkClientWithoutPolicies
BenchmarkClientWithCircuitBreaker
BenchmarkClientWithRetry
BenchmarkClientWithAllPolicies

// Measure throughput
BenchmarkConcurrentRequests/workers=10
BenchmarkConcurrentRequests/workers=100

// Memory allocations
BenchmarkAllocationPerRequest
```

## Project Structure

```
httpx/
├── go.mod                      # Module definition
├── go.sum                      # Dependency checksums
├── README.md                   # User documentation
├── CHANGELOG.md                # Version history
├── client.go                   # Client type, builder pattern
├── transport.go                # Transport interface, default impl
├── request.go                  # Request type, options
├── errors.go                   # Error types and helpers
├── options.go                  # Client options (WithCircuitBreaker, etc)
├── policy/
│   ├── policy.go               # Policy interface, chain executor
│   ├── circuitbreaker.go       # Circuit breaker implementation
│   ├── circuitbreaker_test.go
│   ├── retry.go                # Retry policy
│   ├── retry_test.go
│   ├── timeout.go              # Timeout policy
│   ├── timeout_test.go
│   ├── bulkhead.go             # Bulkhead/rate limiting
│   └── bulkhead_test.go
├── backoff/
│   ├── backoff.go              # Backoff interface
│   ├── exponential.go          # Exponential backoff
│   ├── linear.go               # Linear backoff
│   └── backoff_test.go
├── observability/
│   ├── otel.go                 # OpenTelemetry integration
│   ├── otel_test.go
│   ├── metrics.go              # Prometheus metrics
│   └── metrics_test.go
├── httpxtest/
│   ├── server.go               # Test server utilities
│   ├── assertions.go           # Metric/trace assertions
│   └── mocks.go                # Mock implementations
└── examples/
    ├── basic/                  # Basic usage example
    │   └── main.go
    ├── microservices/          # Microservice communication example
    │   └── main.go
    └── custom_policy/          # Custom policy example
        └── main.go
```

## Dependencies

### Core Dependencies (Go 1.23.0+)

- **stdlib**:
  - `net/http` - HTTP client
  - `context` - Request cancellation and deadlines
  - `sync` - Thread-safe policy state
  - `time` - Timeouts and backoff

### Observability

- `go.opentelemetry.io/otel` (v1.24+) - OTEL SDK
- `go.opentelemetry.io/otel/trace` - Distributed tracing
- `go.opentelemetry.io/otel/metric` - Metrics API
- `github.com/prometheus/client_golang` (v1.19+) - Prometheus client

### Testing

- `github.com/stretchr/testify` (v1.9+) - Assertions and mocks
- `net/http/httptest` (stdlib) - Test HTTP servers

**No legacy dependencies**: Clean slate without outdated libraries like hystrix-go.

## Implementation Phases

### Phase 1: Core Infrastructure
- Transport interface and default implementation
- Policy interface and chain executor
- Client type with builder pattern
- Request type and options system

### Phase 2: Resilience Policies
- Circuit breaker with state machine
- Retry policy with backoff strategies
- Timeout policy with granular controls
- Bulkhead policy with semaphore

### Phase 3: Observability
- OpenTelemetry tracing integration
- Prometheus metrics implementation
- Context propagation
- Metric cardinality control

### Phase 4: Testing
- Unit tests for each policy
- Integration tests with httptest
- httpxtest utilities package
- Benchmarks and performance tests

### Phase 5: Documentation
- README with quick start guide
- GoDoc comments on all exports
- Runnable examples
- Migration guide (from old httpx if applicable)

## Design Decisions

### Why Policy-Based Decorator over Middleware Chain?

**Chosen**: Policy-Based Decorator
**Alternative**: Middleware Chain (like http.Handler pattern)

**Rationale**:
- Policies have richer semantics than middleware (can retry, maintain state)
- Clear separation between transport and resilience concerns
- Easier to test policies in isolation
- More natural API for configuration (builder pattern)

### Why Per-Host Circuit Breakers?

Each target service gets independent circuit breaker state to prevent:
- One failing service opening circuit for all services
- False positives from low-traffic services
- Blast radius limited to specific service

### Why Fail-Fast for Bulkhead?

**Chosen**: Immediate error when capacity exceeded
**Alternative**: Queue requests with timeout

**Rationale**:
- Predictable latency (no queueing delays)
- Simpler implementation (no queue management)
- Clear feedback to caller (explicit capacity error)
- Backpressure propagates to caller for handling

### Why Immutable Client Configuration?

**Chosen**: Configuration set at creation time, immutable after
**Alternative**: Mutable configuration with setters

**Rationale**:
- Thread-safe by design (no concurrent modification)
- Clear lifecycle (create → use → discard)
- Prevents configuration drift in long-running services
- Easier to reason about behavior

## Success Criteria

The httpx package will be considered successful if it:

1. **Prevents cascading failures**: Circuit breaker correctly isolates failing services
2. **Improves reliability**: Automatic retry recovers from transient failures
3. **Provides visibility**: OTEL traces and Prometheus metrics enable debugging
4. **Maintains performance**: Overhead <5% compared to raw net/http for happy path
5. **Easy to use**: Clear API, good defaults, minimal configuration required
6. **Well-tested**: >90% code coverage, integration tests for all policies
7. **Production-ready**: Used in at least one internal microservice

## Future Enhancements (Out of Scope for v1.0)

- Adaptive timeout based on percentile latency
- Request deduplication (same request in-flight)
- Response caching layer
- Rate limiting (client-side)
- Connection health checks (preemptive circuit breaking)
- Dynamic configuration updates (hot reload)
- gRPC support (in addition to HTTP)
- WebSocket support with reconnection logic

## References

- [Release It! by Michael Nygard](https://pragprog.com/titles/mnee2/release-it-second-edition/) - Resilience patterns
- [OpenTelemetry Go SDK](https://opentelemetry.io/docs/languages/go/)
- [Prometheus Go Client](https://github.com/prometheus/client_golang)
- [Hystrix: Latency and Fault Tolerance](https://github.com/Netflix/Hystrix/wiki) - Circuit breaker patterns
- [Go net/http documentation](https://pkg.go.dev/net/http)
