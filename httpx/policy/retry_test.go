package policy_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/seb7887/gofw/httpx/backoff"
	"github.com/seb7887/gofw/httpx/policy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryPolicy_SuccessOnFirstAttempt(t *testing.T) {
	retryPolicy := policy.NewRetryPolicy(policy.RetryConfig{
		MaxAttempts: 3,
		Backoff:     backoff.NewConstantBackoff(10 * time.Millisecond),
	})

	attempts := 0
	executor := func(ctx context.Context, req *http.Request) (*http.Response, error) {
		attempts++
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("success")),
		}, nil
	}

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := retryPolicy.Execute(context.Background(), req, executor)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 1, attempts, "should succeed on first attempt without retry")
}

func TestRetryPolicy_RetriesOnError(t *testing.T) {
	retryPolicy := policy.NewRetryPolicy(policy.RetryConfig{
		MaxAttempts: 3,
		Backoff:     backoff.NewConstantBackoff(10 * time.Millisecond),
	})

	attempts := 0
	executor := func(ctx context.Context, req *http.Request) (*http.Response, error) {
		attempts++
		if attempts < 3 {
			return nil, errors.New("network error")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("success")),
		}, nil
	}

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := retryPolicy.Execute(context.Background(), req, executor)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 3, attempts, "should retry twice before success")
}

func TestRetryPolicy_RetriesOn5xx(t *testing.T) {
	retryPolicy := policy.NewRetryPolicy(policy.RetryConfig{
		MaxAttempts: 3,
		Backoff:     backoff.NewConstantBackoff(10 * time.Millisecond),
	})

	attempts := 0
	executor := func(ctx context.Context, req *http.Request) (*http.Response, error) {
		attempts++
		statusCode := http.StatusServiceUnavailable
		if attempts >= 3 {
			statusCode = http.StatusOK
		}
		return &http.Response{
			StatusCode: statusCode,
			Body:       io.NopCloser(bytes.NewBufferString("")),
		}, nil
	}

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := retryPolicy.Execute(context.Background(), req, executor)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 3, attempts)
}

func TestRetryPolicy_ExhaustsRetries(t *testing.T) {
	retryPolicy := policy.NewRetryPolicy(policy.RetryConfig{
		MaxAttempts: 3,
		Backoff:     backoff.NewConstantBackoff(10 * time.Millisecond),
	})

	attempts := 0
	executor := func(ctx context.Context, req *http.Request) (*http.Response, error) {
		attempts++
		return nil, errors.New("persistent error")
	}

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	_, err := retryPolicy.Execute(context.Background(), req, executor)

	require.Error(t, err)
	assert.Equal(t, 3, attempts, "should attempt exactly MaxAttempts times")
	assert.Contains(t, err.Error(), "max retry attempts exceeded")
}

func TestRetryPolicy_NonIdempotentMethod(t *testing.T) {
	retryPolicy := policy.NewRetryPolicy(policy.RetryConfig{
		MaxAttempts:    3,
		OnlyIdempotent: true,
		Backoff:        backoff.NewConstantBackoff(10 * time.Millisecond),
	})

	attempts := 0
	executor := func(ctx context.Context, req *http.Request) (*http.Response, error) {
		attempts++
		return nil, errors.New("error")
	}

	// POST is non-idempotent
	req, _ := http.NewRequest(http.MethodPost, "http://example.com", nil)
	_, err := retryPolicy.Execute(context.Background(), req, executor)

	require.Error(t, err)
	assert.Equal(t, 1, attempts, "POST should not be retried by default")
}
