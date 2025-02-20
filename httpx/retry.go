package httpx

import "time"

type Retrier interface {
	NextInterval(retry int) time.Duration
}

type RetriableFunc func(retry int) time.Duration

func (f RetriableFunc) NextInterval(retry int) time.Duration {
	return f(retry)
}

type retrier struct {
	backoff Backoff
}

func NewRetrier(backoff Backoff) Retrier {
	return &retrier{backoff: backoff}
}
