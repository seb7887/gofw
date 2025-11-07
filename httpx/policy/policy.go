package policy

import (
	"context"
	"net/http"
)

// Executor is a function that executes an HTTP request.
// It represents the next step in the policy chain (either another policy or the transport).
type Executor func(ctx context.Context, req *http.Request) (*http.Response, error)

// Policy represents a resilience pattern that can be applied to HTTP requests.
// Policies are chained together using the decorator pattern, with each policy
// wrapping the next one in the chain.
//
// A policy can:
// - Execute the next policy/transport by calling next()
// - Short-circuit and return early (e.g., circuit breaker open)
// - Retry by calling next() multiple times
// - Modify the request or response
// - Record metrics and traces
type Policy interface {
	// Execute runs the policy logic around the next executor in the chain.
	// The context can be used for cancellation, timeouts, and passing values.
	// The request represents the HTTP request to be executed.
	// The next function represents the next policy or transport in the chain.
	Execute(ctx context.Context, req *http.Request, next Executor) (*http.Response, error)
}

// Chain creates an executor that chains multiple policies together.
// Policies are applied in order: the first policy wraps the second, which wraps the third, etc.
// The final executor (typically a Transport.Do) is called after all policies have been applied.
//
// Example:
//
//	executor := Chain(
//	    circuitBreakerPolicy,
//	    retryPolicy,
//	    timeoutPolicy,
//	    transport.Do,
//	)
//	resp, err := executor(ctx, req)
func Chain(policies []Policy, final Executor) Executor {
	// Build the chain from the end backwards
	executor := final

	// Wrap each policy around the next one, starting from the last policy
	for i := len(policies) - 1; i >= 0; i-- {
		policy := policies[i]
		next := executor

		// Create a closure that captures the current policy and next executor
		executor = func(ctx context.Context, req *http.Request) (*http.Response, error) {
			return policy.Execute(ctx, req, next)
		}
	}

	return executor
}
