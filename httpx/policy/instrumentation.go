package policy

import (
	"context"
	"net/http"

	"github.com/seb7887/gofw/httpx/observability"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentationPolicy provides OpenTelemetry distributed tracing for HTTP requests.
// It automatically creates spans, propagates trace context, and records request metadata.
type InstrumentationPolicy struct {
	instrumenter *observability.OTELInstrumenter
}

// NewInstrumentationPolicy creates a new instrumentation policy with OTEL support.
func NewInstrumentationPolicy(provider trace.TracerProvider) *InstrumentationPolicy {
	return &InstrumentationPolicy{
		instrumenter: observability.NewOTELInstrumenter(provider),
	}
}

// Execute implements the Policy interface by wrapping the request with an OTEL span.
func (i *InstrumentationPolicy) Execute(ctx context.Context, req *http.Request, next Executor) (*http.Response, error) {
	// Start span and inject trace context into headers
	ctx, span := i.instrumenter.StartSpan(ctx, req)
	defer span.End()

	// Execute request
	resp, err := next(ctx, req)

	// Record response/error in span
	i.instrumenter.EndSpan(span, resp, err)

	return resp, err
}

// Instrumenter returns the underlying OTEL instrumenter.
// This allows policies to add custom attributes and events to the current span.
func (i *InstrumentationPolicy) Instrumenter() *observability.OTELInstrumenter {
	return i.instrumenter
}
