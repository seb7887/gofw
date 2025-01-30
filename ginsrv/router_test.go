package ginsrv

import (
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetupRouter(t *testing.T) {
	// Define routes
	routes := []Route{
		{
			Method: http.MethodGet,
			Path:   "/health",
			Handler: func(c *gin.Context) {
				c.JSON(200, gin.H{"health": "ok"})
			},
		},
	}

	// Set up router with routes
	r := SetupRouter(routes)

	// Create a new HTTP request
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	// Perform the request
	r.ServeHTTP(w, req)

	// Assert the response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"health":"ok"}`, w.Body.String())
}
