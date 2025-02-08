package ginsrv

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/assert/v2"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestErrorFormatterMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		statusCode     int
		expectedBody   string
		expectedStatus int
	}{
		{
			name:           "No error, should pass through",
			statusCode:     http.StatusOK,
			expectedBody:   ``,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Bad request error",
			statusCode:     http.StatusBadRequest,
			expectedBody:   `{"message":"Bad Request"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Not found error",
			statusCode:     http.StatusNotFound,
			expectedBody:   `{"message":"Not Found"}`,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Internal server error",
			statusCode:     http.StatusInternalServerError,
			expectedBody:   `{"message":"Internal Server Error"}`,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(ErrorFormatterMiddleware())

			router.GET("/test", func(c *gin.Context) {
				c.Status(tt.statusCode)
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, tt.expectedBody, w.Body.String())
		})
	}
}
