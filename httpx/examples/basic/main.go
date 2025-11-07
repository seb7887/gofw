package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/seb7887/gofw/httpx"
	"github.com/seb7887/gofw/httpx/backoff"
	"github.com/seb7887/gofw/httpx/policy"
)

func main() {
	// Create a resilient HTTP client with all policies enabled
	client := httpx.NewClient(
		// Base URL for all requests
		httpx.WithBaseURL("https://api.example.com"),

		// Retry policy with exponential backoff
		httpx.WithRetry(policy.RetryConfig{
			MaxAttempts: 3,
			Backoff: &backoff.ExponentialBackoff{
				Initial: 100 * time.Millisecond,
				Max:     2 * time.Second,
				Factor:  2.0,
				Jitter:  true,
			},
		}),

		// Circuit breaker to prevent cascading failures
		httpx.WithCircuitBreaker(policy.CircuitBreakerConfig{
			ErrorThreshold:   50,   // Open circuit if 50% of requests fail
			MinRequests:      10,   // Minimum 10 requests before evaluating
			SleepWindow:      5 * time.Second,
			SuccessThreshold: 2,    // 2 successes to close circuit
		}),

		// Timeout policy
		httpx.WithTimeout(policy.TimeoutConfig{
			Request: 10 * time.Second,
		}),

		// Bulkhead for concurrency limiting
		httpx.WithBulkhead(policy.BulkheadConfig{
			MaxConcurrent: 100,
			PerHost:       true,
		}),
	)

	// Make a simple GET request
	ctx := context.Background()
	resp, err := client.Get(ctx, "/users/123")
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("Status: %d\n", resp.StatusCode)

	// Make a POST request with custom headers
	headers := httpx.Headers{
		"Content-Type":  "application/json",
		"Authorization": "Bearer token123",
	}

	resp2, err := client.Post(ctx, "/users", headers, nil)
	if err != nil {
		log.Fatalf("POST failed: %v", err)
	}
	defer resp2.Body.Close()

	fmt.Printf("Created: %d\n", resp2.StatusCode)

	// Advanced request with per-request options
	resp3, err := client.Do(ctx, &httpx.Request{
		Method:  "POST",
		Path:    "/orders",
		Headers: headers,
		Options: []httpx.RequestOption{
			// Override timeout for this specific request
			httpx.WithRequestTimeout(5 * time.Second),
			// Enable retry for this POST request
			httpx.WithRetryable(true),
		},
	})
	if err != nil {
		log.Fatalf("Advanced request failed: %v", err)
	}
	defer resp3.Body.Close()

	fmt.Printf("Order created: %d\n", resp3.StatusCode)
}
