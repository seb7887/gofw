package httpx_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/seb7887/gofw/httpx"
	"github.com/seb7887/gofw/httpx/httpxtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Get(t *testing.T) {
	// Create mock transport
	mockTransport := &httpxtest.MockTransport{
		Response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("test response")),
		},
	}

	// Create client with mock transport
	client := httpx.NewClient(
		httpx.WithTransport(mockTransport),
		httpx.WithBaseURL("http://example.com"),
	)

	// Make GET request
	ctx := context.Background()
	resp, err := client.Get(ctx, "/test")

	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify request was made
	assert.Equal(t, 1, mockTransport.CallCount)
	lastReq := mockTransport.LastRequest()
	assert.NotNil(t, lastReq)
	assert.Equal(t, http.MethodGet, lastReq.Method)
	assert.Equal(t, "http://example.com/test", lastReq.URL.String())
}

func TestClient_Post(t *testing.T) {
	mockTransport := &httpxtest.MockTransport{
		Response: &http.Response{
			StatusCode: http.StatusCreated,
			Body:       io.NopCloser(bytes.NewBufferString("")),
		},
	}

	client := httpx.NewClient(
		httpx.WithTransport(mockTransport),
		httpx.WithBaseURL("http://example.com"),
	)

	ctx := context.Background()
	body := bytes.NewBufferString(`{"name": "test"}`)
	resp, err := client.Post(ctx, "/users", httpx.Headers{"Content-Type": "application/json"}, body)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Verify request
	lastReq := mockTransport.LastRequest()
	assert.Equal(t, http.MethodPost, lastReq.Method)
	assert.Equal(t, "application/json", lastReq.Header.Get("Content-Type"))
}

func TestClient_WithTestServer(t *testing.T) {
	// Create test server
	server := httpxtest.NewTestServerWithOptions(
		httpxtest.WithStatusCodes(http.StatusOK),
	)
	defer server.Close()

	// Create client
	client := httpx.NewClient(
		httpx.WithBaseURL(server.URL),
	)

	// Make request
	ctx := context.Background()
	resp, err := client.Get(ctx, "/test")

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 1, server.RequestCount())
}
