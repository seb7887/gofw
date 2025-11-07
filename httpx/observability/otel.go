package observability

import (
	"context"
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	instrumentationName = "github.com/seb7887/gofw/httpx"
)

// OTELInstrumenter provides OpenTelemetry instrumentation for HTTP requests.
// It creates spans, injects trace context into headers, and records request metadata.
type OTELInstrumenter struct {
	tracer     trace.Tracer
	propagator propagation.TextMapPropagator
}

// NewOTELInstrumenter creates a new OTEL instrumenter with the given tracer provider.
// If provider is nil, uses the global tracer provider.
func NewOTELInstrumenter(provider trace.TracerProvider) *OTELInstrumenter {
	if provider == nil {
		provider = otel.GetTracerProvider()
	}

	return &OTELInstrumenter{
		tracer:     provider.Tracer(instrumentationName),
		propagator: otel.GetTextMapPropagator(),
	}
}

// StartSpan creates a new span for an HTTP request and returns the updated context.
// The span includes standard HTTP semantic conventions attributes.
func (o *OTELInstrumenter) StartSpan(ctx context.Context, req *http.Request) (context.Context, trace.Span) {
	// Create span with HTTP method as operation name
	spanName := fmt.Sprintf("HTTP %s", req.Method)
	ctx, span := o.tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindClient),
	)

	// Add HTTP semantic convention attributes
	span.SetAttributes(
		attribute.String("http.method", req.Method),
		attribute.String("http.url", req.URL.String()),
		attribute.String("http.scheme", req.URL.Scheme),
		attribute.String("http.host", req.URL.Host),
		attribute.String("http.target", req.URL.Path),
	)

	if req.URL.RawQuery != "" {
		span.SetAttributes(attribute.String("http.query", req.URL.RawQuery))
	}

	// Inject trace context into request headers
	o.propagator.Inject(ctx, propagation.HeaderCarrier(req.Header))

	return ctx, span
}

// EndSpan completes the span with response information.
// Records status code, errors, and marks the span as error if applicable.
func (o *OTELInstrumenter) EndSpan(span trace.Span, resp *http.Response, err error) {
	if err != nil {
		// Record error
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else if resp != nil {
		// Record response status
		span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

		// Set span status based on HTTP status code
		if resp.StatusCode >= 400 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", resp.StatusCode))
		} else {
			span.SetStatus(codes.Ok, "")
		}
	}

	span.End()
}

// AddRetryAttribute adds retry count information to the span.
func (o *OTELInstrumenter) AddRetryAttribute(span trace.Span, retryCount int) {
	if retryCount > 0 {
		span.SetAttributes(attribute.Int("http.retry_count", retryCount))
	}
}

// AddCircuitBreakerAttribute adds circuit breaker state to the span.
func (o *OTELInstrumenter) AddCircuitBreakerAttribute(span trace.Span, state string) {
	span.SetAttributes(attribute.String("http.circuit_breaker_state", state))
}

// AddPolicyEvent adds an event to the span indicating a policy action.
// Useful for recording retry attempts, circuit breaker opens, etc.
func (o *OTELInstrumenter) AddPolicyEvent(span trace.Span, eventName string, attrs ...attribute.KeyValue) {
	span.AddEvent(eventName, trace.WithAttributes(attrs...))
}
