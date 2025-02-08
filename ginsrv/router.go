package ginsrv

import (
	"github.com/gin-gonic/gin"
	"os"
)

type Route struct {
	Method  string
	Path    string
	Handler gin.HandlerFunc
}

func SetupRouter(routes []Route, middlewares ...gin.HandlerFunc) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	if os.Getenv("ENV") == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Apply middlewares in reverse order
	for i := len(middlewares) - 1; i >= 0; i-- {
		router.Use(middlewares[i])
	}

	// Generate all the routes
	for _, route := range routes {
		router.Handle(route.Method, route.Path, route.Handler)
	}

	return router
}
