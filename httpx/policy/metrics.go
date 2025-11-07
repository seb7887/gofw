package policy

import (
	"context"
	"net/http"
	"time"

	"github.com/seb7887/gofw/httpx/observability"
)

// MetricsPolicy provides Prometheus metrics collection for HTTP requests.
// It tracks request duration, active requests, and integrates with other policies
// to record circuit breaker states, retries, and bulkhead rejections.
type MetricsPolicy struct {
	collector *observability.MetricsCollector
}

// NewMetricsPolicy creates a new metrics policy with the given collector.
func NewMetricsPolicy(collector *observability.MetricsCollector) *MetricsPolicy {
	return &MetricsPolicy{
		collector: collector,
	}
}

// Execute implements the Policy interface by recording request metrics.
func (m *MetricsPolicy) Execute(ctx context.Context, req *http.Request, next Executor) (*http.Response, error) {
	host := observability.NormalizeHost(req.URL.Host)

	// Increment active requests
	m.collector.IncrementActiveRequests(host)
	defer m.collector.DecrementActiveRequests(host)

	// Record start time
	startTime := time.Now()

	// Execute request
	resp, err := next(ctx, req)

	// Record duration
	duration := time.Since(startTime)

	// Record metrics
	if resp != nil {
		m.collector.RecordRequestDuration(req.Method, host, resp.StatusCode, duration)
	} else if err != nil {
		// Record as 0 status code for errors
		m.collector.RecordRequestDuration(req.Method, host, 0, duration)
	}

	return resp, err
}

// Collector returns the underlying metrics collector.
// This allows other policies to record their specific metrics.
func (m *MetricsPolicy) Collector() *observability.MetricsCollector {
	return m.collector
}
