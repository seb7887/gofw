package observability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MetricsCollector provides Prometheus metrics collection for HTTP requests.
type MetricsCollector struct {
	requestDuration      *prometheus.HistogramVec
	circuitBreakerState  *prometheus.GaugeVec
	circuitBreakerFails  *prometheus.CounterVec
	retryAttempts        *prometheus.CounterVec
	activeRequests       *prometheus.GaugeVec
	bulkheadRejections   *prometheus.CounterVec
}

// NewMetricsCollector creates a new Prometheus metrics collector.
// If registry is nil, uses the default Prometheus registry.
func NewMetricsCollector(registry prometheus.Registerer) *MetricsCollector {
	if registry == nil {
		registry = prometheus.DefaultRegisterer
	}

	factory := promauto.With(registry)

	return &MetricsCollector{
		requestDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "http_client_request_duration_seconds",
				Help: "HTTP client request duration in seconds",
				Buckets: []float64{
					0.001, // 1ms
					0.005, // 5ms
					0.01,  // 10ms
					0.05,  // 50ms
					0.1,   // 100ms
					0.5,   // 500ms
					1.0,   // 1s
					2.0,   // 2s
					5.0,   // 5s
					10.0,  // 10s
				},
			},
			[]string{"method", "status_code", "host"},
		),

		circuitBreakerState: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "http_client_circuit_breaker_state",
				Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
			},
			[]string{"host"},
		),

		circuitBreakerFails: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_client_circuit_breaker_failures_total",
				Help: "Total number of circuit breaker failures",
			},
			[]string{"host"},
		),

		retryAttempts: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_client_retries_total",
				Help: "Total number of retry attempts",
			},
			[]string{"method", "host", "reason"},
		),

		activeRequests: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "http_client_active_requests",
				Help: "Number of active HTTP requests",
			},
			[]string{"host"},
		),

		bulkheadRejections: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_client_rejected_requests_total",
				Help: "Total number of requests rejected by bulkhead",
			},
			[]string{"host"},
		),
	}
}

// RecordRequestDuration records the duration of an HTTP request.
func (m *MetricsCollector) RecordRequestDuration(method, host string, statusCode int, duration time.Duration) {
	m.requestDuration.WithLabelValues(
		method,
		strconv.Itoa(statusCode),
		host,
	).Observe(duration.Seconds())
}

// SetCircuitBreakerState sets the circuit breaker state metric.
// state: 0=closed, 1=open, 2=half-open
func (m *MetricsCollector) SetCircuitBreakerState(host string, state int) {
	m.circuitBreakerState.WithLabelValues(host).Set(float64(state))
}

// IncrementCircuitBreakerFailures increments the circuit breaker failure counter.
func (m *MetricsCollector) IncrementCircuitBreakerFailures(host string) {
	m.circuitBreakerFails.WithLabelValues(host).Inc()
}

// IncrementRetryAttempts increments the retry attempt counter.
// reason: "network_error", "5xx", "429", "custom"
func (m *MetricsCollector) IncrementRetryAttempts(method, host, reason string) {
	m.retryAttempts.WithLabelValues(method, host, reason).Inc()
}

// IncrementActiveRequests increments the active requests gauge.
func (m *MetricsCollector) IncrementActiveRequests(host string) {
	m.activeRequests.WithLabelValues(host).Inc()
}

// DecrementActiveRequests decrements the active requests gauge.
func (m *MetricsCollector) DecrementActiveRequests(host string) {
	m.activeRequests.WithLabelValues(host).Dec()
}

// IncrementBulkheadRejections increments the bulkhead rejection counter.
func (m *MetricsCollector) IncrementBulkheadRejections(host string) {
	m.bulkheadRejections.WithLabelValues(host).Inc()
}

// NormalizeHost normalizes a host string for use in metrics.
// Strips default ports to reduce cardinality.
func NormalizeHost(host string) string {
	// TODO: Strip default ports (":80", ":443")
	// For now, return as-is
	return host
}

// StatusCodeToReason converts an HTTP status code to a retry reason.
func StatusCodeToReason(req *http.Request, statusCode int) string {
	if statusCode == 429 {
		return "429"
	}
	if statusCode >= 500 {
		return "5xx"
	}
	return "unknown"
}
