package httpx

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/seb7887/gofw/httpx/observability"
	"github.com/seb7887/gofw/httpx/policy"
	"go.opentelemetry.io/otel/trace"
)

// ClientOption configures a Client during creation.
type ClientOption interface {
	apply(*Client)
}

// funcClientOption wraps a function to implement ClientOption
type funcClientOption struct {
	f func(*Client)
}

func (fo *funcClientOption) apply(c *Client) {
	fo.f(c)
}

// WithHTTPClient sets a custom http.Client to use as the underlying transport.
// This allows full control over connection pooling, TLS configuration, proxies, etc.
//
// Example:
//
//	client := httpx.NewClient(
//	    httpx.WithHTTPClient(&http.Client{
//	        Transport: &http.Transport{
//	            MaxIdleConns:        100,
//	            MaxIdleConnsPerHost: 10,
//	        },
//	    }),
//	)
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return &funcClientOption{
		f: func(c *Client) {
			c.transport = NewDefaultTransportWithClient(httpClient)
		},
	}
}

// WithTransport sets a custom Transport implementation.
// This is useful for testing with mock transports.
func WithTransport(transport Transport) ClientOption {
	return &funcClientOption{
		f: func(c *Client) {
			c.transport = transport
		},
	}
}

// WithBaseURL sets the base URL that will be prepended to all request paths.
// The path from each request will be joined with this base URL.
//
// Example:
//
//	client := httpx.NewClient(
//	    httpx.WithBaseURL("http://service-b:8080"),
//	)
//	// client.Get(ctx, "/users") will request http://service-b:8080/users
func WithBaseURL(baseURL string) ClientOption {
	return &funcClientOption{
		f: func(c *Client) {
			c.baseURL = baseURL
		},
	}
}

// WithPolicy adds a custom policy to the client's policy chain.
// Policies are applied in the order they are added.
func WithPolicy(p policy.Policy) ClientOption {
	return &funcClientOption{
		f: func(c *Client) {
			c.policies = append(c.policies, p)
		},
	}
}

// WithCircuitBreaker adds a circuit breaker policy to prevent cascading failures.
// The circuit breaker tracks error rates and opens when thresholds are exceeded.
//
// Example:
//
//	client := httpx.NewClient(
//	    httpx.WithCircuitBreaker(httpx.CircuitBreakerConfig{
//	        ErrorThreshold: 50,
//	        MinRequests: 10,
//	        SleepWindow: 5 * time.Second,
//	        SuccessThreshold: 2,
//	    }),
//	)
func WithCircuitBreaker(config policy.CircuitBreakerConfig) ClientOption {
	return &funcClientOption{
		f: func(c *Client) {
			c.policies = append(c.policies, policy.NewCircuitBreakerPolicy(config))
		},
	}
}

// WithRetry adds a retry policy with configurable backoff strategies.
// Failed requests will be retried according to the configuration.
//
// Example:
//
//	client := httpx.NewClient(
//	    httpx.WithRetry(httpx.RetryConfig{
//	        MaxAttempts: 3,
//	        Backoff: httpx.NewExponentialBackoff(),
//	    }),
//	)
func WithRetry(config policy.RetryConfig) ClientOption {
	return &funcClientOption{
		f: func(c *Client) {
			c.policies = append(c.policies, policy.NewRetryPolicy(config))
		},
	}
}

// WithTimeout adds timeout controls at multiple levels (connection, request, etc).
//
// Example:
//
//	client := httpx.NewClient(
//	    httpx.WithTimeout(httpx.TimeoutConfig{
//	        Request: 10 * time.Second,
//	    }),
//	)
func WithTimeout(config policy.TimeoutConfig) ClientOption {
	return &funcClientOption{
		f: func(c *Client) {
			c.policies = append(c.policies, policy.NewTimeoutPolicy(config))
		},
	}
}

// WithBulkhead adds concurrency limiting to prevent resource exhaustion.
//
// Example:
//
//	client := httpx.NewClient(
//	    httpx.WithBulkhead(httpx.BulkheadConfig{
//	        MaxConcurrent: 100,
//	        PerHost: true,
//	    }),
//	)
func WithBulkhead(config policy.BulkheadConfig) ClientOption {
	return &funcClientOption{
		f: func(c *Client) {
			c.policies = append(c.policies, policy.NewBulkheadPolicy(config))
		},
	}
}

// WithOTEL enables OpenTelemetry distributed tracing.
// The instrumentation policy should typically be added first in the policy chain
// to ensure all subsequent policies are traced.
//
// Example:
//
//	client := httpx.NewClient(
//	    httpx.WithOTEL(otelProvider),
//	    httpx.WithRetry(...),
//	    httpx.WithCircuitBreaker(...),
//	)
func WithOTEL(provider trace.TracerProvider) ClientOption {
	return &funcClientOption{
		f: func(c *Client) {
			c.policies = append(c.policies, policy.NewInstrumentationPolicy(provider))
		},
	}
}

// WithMetrics enables Prometheus metrics collection.
// The metrics policy should typically be added early in the policy chain
// (after instrumentation if using OTEL) to capture metrics from all policies.
//
// Example:
//
//	registry := prometheus.NewRegistry()
//	client := httpx.NewClient(
//	    httpx.WithOTEL(otelProvider),
//	    httpx.WithMetrics(registry),
//	    httpx.WithRetry(...),
//	    httpx.WithCircuitBreaker(...),
//	)
func WithMetrics(registry prometheus.Registerer) ClientOption {
	return &funcClientOption{
		f: func(c *Client) {
			collector := observability.NewMetricsCollector(registry)
			c.policies = append(c.policies, policy.NewMetricsPolicy(collector))
		},
	}
}
