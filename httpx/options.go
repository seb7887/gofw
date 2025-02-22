package httpx

import (
	"net/http"
	"time"
)

type Option func(*client)

func WithHTTPTimeout(timeout time.Duration) Option {
	return func(c *client) {
		c.timeout = timeout
	}
}

func WithRetryCount(retryCount int) Option {
	return func(c *client) {
		c.retryCount = retryCount
	}
}

func WithRetrier(retrier Retrier) Option {
	return func(c *client) {
		c.retrier = retrier
	}
}

func WithHTTPClient(c *http.Client) Option {
	return func(c *client) {
		c.c = c
	}
}
