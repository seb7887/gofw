# httpx - Resilient HTTP Client for Go

**httpx** is a production-ready HTTP client library for Go microservices that implements comprehensive resilience patterns including circuit breakers, intelligent retry logic, granular timeouts, and bulkhead isolation. It provides deep observability through OpenTelemetry distributed tracing and Prometheus metrics.

## Features

### Resilience Policies

- **Circuit Breaker**: Prevents cascading failures with per-host state management (closed/open/half-open)
- **Retry with Backoff**: Configurable retry strategies (exponential, linear, constant) with jitter support
- **Timeout Control**: Fine-grained timeout management at request level
- **Bulkhead Pattern**: Semaphore-based concurrency limiting for resource protection

### Observability

- **OpenTelemetry Integration**: Automatic distributed tracing with W3C Trace Context propagation
- **Prometheus Metrics**: Request duration, circuit breaker state, retry attempts, active requests, and rejections
- **Structured Attributes**: HTTP semantic conventions for spans and metrics

### Design

- **Policy-Based Architecture**: Composable decorator pattern for flexible policy chains
- **Builder Pattern API**: Fluent configuration with sensible defaults
- **Thread-Safe**: All policies designed for concurrent use
- **Type-Safe**: Leverages Go generics and strong typing

## Installation

```bash
go get github.com/seb7887/gofw/httpx
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/seb7887/gofw/httpx"
    "github.com/seb7887/gofw/httpx/policy"
)

func main() {
    // Create client with resilience policies
    client := httpx.NewClient(
        httpx.WithBaseURL("https://api.example.com"),
        httpx.WithRetry(policy.RetryConfig{
            MaxAttempts: 3,
        }),
        httpx.WithCircuitBreaker(policy.CircuitBreakerConfig{
            ErrorThreshold: 50,
            MinRequests:    10,
        }),
        httpx.WithTimeout(policy.TimeoutConfig{
            Request: 10 * time.Second,
        }),
    )

    // Make requests
    ctx := context.Background()
    resp, err := client.Get(ctx, "/users/123")
    if err != nil {
        log.Fatal(err)
    }
    defer resp.Body.Close()
}
```

### With Full Observability

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "go.opentelemetry.io/otel"
)

// Create registry for metrics
registry := prometheus.NewRegistry()

// Get OTEL tracer provider
tracerProvider := otel.GetTracerProvider()

// Create client with observability
client := httpx.NewClient(
    httpx.WithBaseURL("https://api.example.com"),

    // Observability (add these first)
    httpx.WithOTEL(tracerProvider),
    httpx.WithMetrics(registry),

    // Resilience policies
    httpx.WithRetry(policy.RetryConfig{
        MaxAttempts: 3,
    }),
    httpx.WithCircuitBreaker(policy.CircuitBreakerConfig{
        ErrorThreshold: 50,
    }),
)
```

## Policies

### Circuit Breaker

Implements a state machine to prevent cascading failures:

```go
httpx.WithCircuitBreaker(policy.CircuitBreakerConfig{
    ErrorThreshold:   50,              // % of errors to trigger open state
    MinRequests:      10,              // Min requests before evaluating
    SleepWindow:      5 * time.Second, // Time in open before half-open
    SuccessThreshold: 2,               // Successes in half-open to close
})
```

**State Transitions:**
- **Closed** → **Open**: When error rate exceeds threshold
- **Open** → **Half-Open**: After sleep window expires
- **Half-Open** → **Closed**: After success threshold met
- **Half-Open** → **Open**: On any failure

**Per-Host Isolation**: Each target host has independent circuit breaker state.

### Retry Policy

Automatic retry with configurable backoff strategies:

```go
import "github.com/seb7887/gofw/httpx/backoff"

httpx.WithRetry(policy.RetryConfig{
    MaxAttempts: 3,
    Backoff: &backoff.ExponentialBackoff{
        Initial: 100 * time.Millisecond,
        Max:     2 * time.Second,
        Factor:  2.0,
        Jitter:  true,
    },
    OnlyIdempotent: true, // Only retry GET, PUT, DELETE (default: true)
})
```

**Backoff Strategies:**
- **Exponential**: `initial * (factor ^ retry)` with optional jitter
- **Linear**: `interval * (retry + 1)`
- **Constant**: Fixed interval

**Default Retry Conditions:**
- Network errors
- HTTP 5xx status codes
- HTTP 429 (rate limit)
- Idempotent methods only (GET, PUT, DELETE, HEAD, OPTIONS)

### Timeout Policy

Granular timeout control:

```go
httpx.WithTimeout(policy.TimeoutConfig{
    Request: 10 * time.Second, // Total request timeout
})
```

Timeouts are enforced via context deadlines and propagate cancellation through the policy chain.

### Bulkhead Policy

Concurrency limiting to prevent resource exhaustion:

```go
httpx.WithBulkhead(policy.BulkheadConfig{
    MaxConcurrent: 100,  // Max concurrent requests
    PerHost:       true, // Per-host isolation (default)
})
```

**Behavior:**
- **Fail-fast**: Returns error immediately if capacity exceeded
- **Per-host**: Each service has independent semaphore
- **No queueing**: Predictable latency

## Per-Request Options

Override client policies for specific requests:

```go
resp, err := client.Do(ctx, &httpx.Request{
    Method:  "POST",
    Path:    "/orders",
    Headers: httpx.Headers{"Content-Type": "application/json"},
    Options: []httpx.RequestOption{
        httpx.WithRequestTimeout(5 * time.Second),  // Override timeout
        httpx.WithRetryable(true),                  // Enable retry for POST
        httpx.WithoutCircuitBreaker(),              // Bypass circuit breaker
    },
})
```

## Observability

### OpenTelemetry Tracing

Automatic span creation with HTTP semantic conventions:

```go
import "go.opentelemetry.io/otel"

provider := otel.GetTracerProvider()
client := httpx.NewClient(
    httpx.WithOTEL(provider),
    // other options...
)
```

**Span Attributes:**
- `http.method`: Request method
- `http.url`: Full URL
- `http.status_code`: Response status
- `peer.service`: Target service
- `http.retry_count`: Number of retries
- `http.circuit_breaker_state`: Circuit breaker state

**Context Propagation**: W3C Trace Context headers injected automatically.

### Prometheus Metrics

Comprehensive metrics for monitoring:

```go
import "github.com/prometheus/client_golang/prometheus"

registry := prometheus.NewRegistry()
client := httpx.NewClient(
    httpx.WithMetrics(registry),
    // other options...
)
```

**Exposed Metrics:**

| Metric | Type | Description | Labels |
|--------|------|-------------|--------|
| `http_client_request_duration_seconds` | Histogram | Request duration | method, status_code, host |
| `http_client_circuit_breaker_state` | Gauge | Circuit state (0/1/2) | host |
| `http_client_circuit_breaker_failures_total` | Counter | Circuit breaker failures | host |
| `http_client_retries_total` | Counter | Retry attempts | method, host, reason |
| `http_client_active_requests` | Gauge | Active requests | host |
| `http_client_rejected_requests_total` | Counter | Bulkhead rejections | host |

## Testing

### Using Mock Transport

```go
import "github.com/seb7887/gofw/httpx/httpxtest"

func TestMyCode(t *testing.T) {
    mockTransport := &httpxtest.MockTransport{
        Response: &http.Response{
            StatusCode: http.StatusOK,
            Body:       io.NopCloser(bytes.NewBufferString("test")),
        },
    }

    client := httpx.NewClient(
        httpx.WithTransport(mockTransport),
    )

    // Test your code...

    // Verify requests
    assert.Equal(t, 1, mockTransport.CallCount)
    lastReq := mockTransport.LastRequest()
    assert.Equal(t, http.MethodGet, lastReq.Method)
}
```

### Using Test Server

```go
import "github.com/seb7887/gofw/httpx/httpxtest"

func TestWithRealServer(t *testing.T) {
    server := httpxtest.NewTestServerWithOptions(
        httpxtest.WithLatency(100 * time.Millisecond),
        httpxtest.WithFailureRate(0.3), // 30% failure rate
        httpxtest.WithStatusCodes(http.StatusOK, http.StatusServiceUnavailable),
    )
    defer server.Close()

    client := httpx.NewClient(
        httpx.WithBaseURL(server.URL),
        httpx.WithRetry(policy.RetryConfig{MaxAttempts: 3}),
    )

    // Test with retry...
}
```

### Metric Assertions

```go
import "github.com/seb7887/gofw/httpx/httpxtest"

func TestMetrics(t *testing.T) {
    registry := prometheus.NewRegistry()

    // ... make requests ...

    httpxtest.AssertMetricValueWithLabels(t, registry,
        "http_client_retries_total",
        map[string]string{"method": "GET", "reason": "5xx"},
        3.0,
    )
}
```

## Architecture

httpx uses a **Policy-Based Decorator Pattern**:

```
┌─────────────────────────────────────────┐
│          Client API (Builder)            │
└─────────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────┐
│         Policy Chain (Decorator)         │
│  OTEL → Metrics → CB → Retry → Timeout  │
└─────────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────┐
│      Transport (HTTP Execution)          │
└─────────────────────────────────────────┘
```

Each policy:
- Implements the `Policy` interface
- Can execute the next policy in the chain
- Can short-circuit (circuit breaker open)
- Can retry (retry policy)
- Can modify requests/responses
- Records metrics and traces

## Requirements

- **Go**: 1.23.0+ (uses generics and modern standard library)
- **Dependencies**:
  - `go.opentelemetry.io/otel` v1.38+
  - `github.com/prometheus/client_golang` v1.23+
  - `github.com/stretchr/testify` v1.10+ (testing only)

## Examples

See the [examples](./examples/) directory for complete working examples:

- [basic](./examples/basic/): Simple client with all policies
- [microservices](./examples/microservices/): Service-to-service communication (TODO)
- [custom_policy](./examples/custom_policy/): Implementing custom policies (TODO)

## Design Documentation

For detailed design rationale, architecture decisions, and implementation details, see:

- [Design Document](../docs/plans/2025-11-05-httpx-resilient-client-design.md)

## License

MIT

## Contributing

This package is part of the [gofw](https://github.com/seb7887/gofw) framework. See the main repository for contribution guidelines.
