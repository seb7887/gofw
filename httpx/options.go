package httpx

import (
	"net/http"

	"github.com/seb7887/gofw/httpx/policy"
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

// TODO: These options will be implemented when we create the actual policies

// WithCircuitBreaker adds a circuit breaker policy to prevent cascading failures.
// The circuit breaker tracks error rates and opens when thresholds are exceeded.
//
// NOTE: Not yet implemented - will be added with CircuitBreakerPolicy
// func WithCircuitBreaker(config CircuitBreakerConfig) ClientOption {
//     return &funcClientOption{
//         f: func(c *Client) {
//             // TODO: Create and add CircuitBreakerPolicy
//         },
//     }
// }

// WithRetry adds a retry policy with configurable backoff strategies.
// Failed requests will be retried according to the configuration.
//
// NOTE: Not yet implemented - will be added with RetryPolicy
// func WithRetry(config RetryConfig) ClientOption {
//     return &funcClientOption{
//         f: func(c *Client) {
//             // TODO: Create and add RetryPolicy
//         },
//     }
// }

// WithTimeout adds timeout controls at multiple levels (connection, request, etc).
//
// NOTE: Not yet implemented - will be added with TimeoutPolicy
// func WithTimeout(config TimeoutConfig) ClientOption {
//     return &funcClientOption{
//         f: func(c *Client) {
//             // TODO: Create and add TimeoutPolicy
//         },
//     }
// }

// WithBulkhead adds concurrency limiting to prevent resource exhaustion.
//
// NOTE: Not yet implemented - will be added with BulkheadPolicy
// func WithBulkhead(maxConcurrent int) ClientOption {
//     return &funcClientOption{
//         f: func(c *Client) {
//             // TODO: Create and add BulkheadPolicy
//         },
//     }
// }

// WithOTEL enables OpenTelemetry distributed tracing.
//
// NOTE: Not yet implemented - will be added with observability integration
// func WithOTEL(provider trace.TracerProvider) ClientOption {
//     return &funcClientOption{
//         f: func(c *Client) {
//             // TODO: Configure OTEL instrumentation
//         },
//     }
// }

// WithMetrics enables Prometheus metrics collection.
//
// NOTE: Not yet implemented - will be added with metrics integration
// func WithMetrics(registry *prometheus.Registry) ClientOption {
//     return &funcClientOption{
//         f: func(c *Client) {
//             // TODO: Configure Prometheus metrics
//         },
//     }
// }
