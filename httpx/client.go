package httpx

import (
	"io"
	"net/http"
)

type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client interface {
	Get(url string, headers http.Header) (*http.Response, error)
	Post(url string, headers http.Header, body io.Reader) (*http.Response, error)
	Put(url string, headers http.Header, body io.Reader) (*http.Response, error)
	Delete(url string, headers http.Header) (*http.Response, error)
	Patch(url string, headers http.Header, body io.Reader) (*http.Response, error)
	Do(req *http.Request) (*http.Response, error)
	AddPlugin(p Plugin)
}
