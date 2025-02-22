package hystrix

import (
	"bytes"
	"errors"
	"github.com/afex/hystrix-go/hystrix"
	metricCollector "github.com/afex/hystrix-go/hystrix/metric_collector"
	"github.com/afex/hystrix-go/plugins"
	"github.com/seb7887/gofw/httpx"
	"io"
	"net/http"
	"time"
)

type Client struct {
	c                      httpx.Client
	timeout                time.Duration
	hystrixTimeout         time.Duration
	hystrixCommandName     string
	maxConcurrentRequests  int
	requestVolumeThreshold int
	sleepWindow            int
	errorPercentThreshold  int
	retryCount             int
	retrier                httpx.Retrier
	fallbackFunc           func(err error) error
	statsD                 *plugins.StatsdCollectorConfig
}

const (
	_defaultHystrixRetryCount      = 0
	_defaultHTTPTimeout            = 30 * time.Second
	_defaultHystrixTimeout         = 30 * time.Second
	_defaultMaxConcurrentRequests  = 100
	_defaultErrorPercentThreshold  = 25
	_defaultSleepWindow            = 10
	_defaultRequestVolumeThreshold = 10

	_maxUint = ^uint(0)
	_maxInt  = int(_maxUint >> 1)
)

var (
	_      httpx.Client = (*Client)(nil)
	Err5xx              = errors.New("server returned 5xx status code")
)

func NewClient(opts ...Option) *Client {
	c := &Client{
		c:                      httpx.NewClient(),
		timeout:                _defaultHTTPTimeout,
		hystrixTimeout:         _defaultHystrixTimeout,
		maxConcurrentRequests:  _defaultMaxConcurrentRequests,
		errorPercentThreshold:  _defaultErrorPercentThreshold,
		sleepWindow:            _defaultSleepWindow,
		requestVolumeThreshold: _defaultRequestVolumeThreshold,
		retryCount:             _defaultHystrixRetryCount,
		retrier:                httpx.NewNoRetrier(),
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.statsD != nil {
		c, err := plugins.InitializeStatsdCollector(c.statsD)
		if err != nil {
			panic(err)
		}
		metricCollector.Registry.Register(c.NewStatsdCollector)
	}

	hystrix.ConfigureCommand(c.hystrixCommandName, hystrix.CommandConfig{
		Timeout:                durationToInt(c.hystrixTimeout, time.Millisecond),
		MaxConcurrentRequests:  c.maxConcurrentRequests,
		RequestVolumeThreshold: c.requestVolumeThreshold,
		SleepWindow:            c.sleepWindow,
		ErrorPercentThreshold:  c.errorPercentThreshold,
	})

	return c
}

func (c *Client) Get(url string, headers http.Header) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Join(err, errors.New("GET - request creation failed"))
	}

	req.Header = headers

	return c.c.Do(req)
}

func (c *Client) Post(url string, headers http.Header, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, errors.Join(err, errors.New("POST - request creation failed"))
	}

	req.Header = headers
	return c.c.Do(req)
}

func (c *Client) Put(url string, headers http.Header, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPut, url, body)
	if err != nil {
		return nil, errors.Join(err, errors.New("PUT - request creation failed"))
	}

	req.Header = headers
	return c.c.Do(req)
}

func (c *Client) Delete(url string, headers http.Header) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return nil, errors.Join(err, errors.New("DELETE - request creation failed"))
	}

	req.Header = headers
	return c.c.Do(req)
}

func (c *Client) Patch(url string, headers http.Header, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPatch, url, body)
	if err != nil {
		return nil, errors.Join(err, errors.New("PATCH - request creation failed"))
	}

	req.Header = headers
	return c.c.Do(req)
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	var (
		resp       *http.Response
		err        error
		bodyReader *bytes.Reader
	)

	if req.Body != nil {
		reqData, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(reqData)
		req.Body = io.NopCloser(bodyReader)
	}

	for i := 0; i < c.retryCount; i++ {
		if resp != nil {
			_ = resp.Body.Close()
		}

		err = hystrix.Do(c.hystrixCommandName, func() error {
			resp, err = c.c.Do(req)
			if bodyReader != nil {
				_, _ = bodyReader.Seek(0, 0)
			}

			if err != nil {
				return err
			}

			if resp.StatusCode >= http.StatusInternalServerError {
				return Err5xx
			}
			return nil
		}, c.fallbackFunc)

		if err == nil {
			backoffTime := c.retrier.NextInterval(i)
			time.Sleep(backoffTime)
			continue
		}

		break
	}

	if errors.Is(err, Err5xx) {
		return resp, nil
	}

	return resp, err
}

func (c *Client) AddPlugin(plugin httpx.Plugin) {
	c.c.AddPlugin(plugin)
}

func durationToInt(duration, unit time.Duration) int {
	durationAsNumber := duration / unit

	if int64(durationAsNumber) > int64(_maxInt) {
		// Returning max possible value seems like best possible solution here
		// the alternative is to panic as there is no way of returning an error
		// without changing the NewClient API
		return _maxInt
	}
	return int(durationAsNumber)
}
