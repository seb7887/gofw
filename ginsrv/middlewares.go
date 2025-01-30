package ginsrv

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func ErrorFormatterMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if c.Writer.Status() >= http.StatusBadRequest {
			c.JSON(c.Writer.Status(), gin.H{
				"message": http.StatusText(c.Writer.Status()),
			})
		}
	}
}
