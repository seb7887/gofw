package httpx

import (
	"bytes"
	"errors"
	"go.uber.org/multierr"
	"io"
	"net/http"
	"time"
)

type client struct {
	c          Doer
	timeout    time.Duration
	retryCount int
	plugins    []Plugin
}

const (
	_defaultTimeout    = time.Second * 30
	_defaultRetryCount = 0
)

var _ Client = (*client)(nil)

func NewClient(opts ...Option) Client {
	c := &client{
		timeout:    _defaultTimeout,
		retryCount: _defaultRetryCount,
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.c == nil {
		c.c = &http.Client{
			Timeout: c.timeout,
		}
	}

	return c
}

func (c *client) AddPlugin(plugin Plugin) {
	c.plugins = append(c.plugins, plugin)
}

func (c *client) Get(url string, headers http.Header) (*http.Response, error) {
	var response *http.Response
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return response, errors.Join(err, errors.New("GET - request creation failed"))
	}

	request.Header = headers

	return c.Do(request)
}

func (c *client) Post(url string, headers http.Header, body io.Reader) (*http.Response, error) {
	var response *http.Response
	request, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return response, errors.Join(err, errors.New("POST - request creation failed"))
	}

	request.Header = headers
	return c.Do(request)
}

func (c *client) Put(url string, headers http.Header, body io.Reader) (*http.Response, error) {
	var response *http.Response
	request, err := http.NewRequest(http.MethodPut, url, body)
	if err != nil {
		return response, errors.Join(err, errors.New("PUT - request creation failed"))
	}

	request.Header = headers
	return c.Do(request)
}

func (c *client) Patch(url string, headers http.Header, body io.Reader) (resp *http.Response, err error) {
	var response *http.Response
	request, err := http.NewRequest(http.MethodPatch, url, body)
	if err != nil {
		return response, errors.Join(err, errors.New("PATCH - request creation failed"))
	}

	request.Header = headers
	return c.Do(request)
}

func (c *client) Delete(url string, headers http.Header) (*http.Response, error) {
	var response *http.Response
	request, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return response, errors.Join(err, errors.New("DELETE - request creation failed"))
	}

	request.Header = headers
	return c.Do(request)
}

func (c *client) Do(request *http.Request) (*http.Response, error) {
	var bodyReader *bytes.Reader

	if request.Body != nil {
		reqData, err := io.ReadAll(request.Body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(reqData)
		request.Body = io.NopCloser(bytes.NewBuffer(reqData))
	}

	var (
		multiErr error
		response *http.Response
	)

	for i := 0; i <= c.retryCount; i++ {
		if response != nil {
			response.Body.Close()
		}

		c.reportRequestStart(request)
		response, err := c.c.Do(request)
		if bodyReader != nil {
			// Reset the body reader after the request since at this point it's already read
			_, _ = bodyReader.Seek(0, 0)
		}

		if err != nil {
			multiErr = multierr.Append(multiErr, err)
			c.reportError(request, err)
			backOffTime := c.retrier.NextInterval(i)
			time.Sleep(backOffTime)
			continue
		}
		c.reportRequestEnd(request, response)

		if response.StatusCode >= http.StatusInternalServerError {
			backOffTime := c.retrier.NextInterval(i)
			time.Sleep(backOffTime)
			continue
		}

		multiErr = nil
		break
	}

	return response, multiErr
}

func (c *client) reportRequestStart(request *http.Request) {
	for _, plugin := range c.plugins {
		plugin.OnRequestStart(request)
	}
}

func (c *client) reportError(request *http.Request, err error) {
	for _, plugin := range c.plugins {
		plugin.OnError(request, err)
	}
}

func (c *client) reportRequestEnd(request *http.Request, response *http.Response) {
	for _, plugin := range c.plugins {
		plugin.OnRequestEnd(request, response)
	}
}
