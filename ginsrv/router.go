package ginsrv

import "github.com/gin-gonic/gin"

type Route struct {
	Method  string
	Path    string
	Handler gin.HandlerFunc
}

func SetupRouter(routes []Route, middlewares ...gin.HandlerFunc) *gin.Engine {
	router := gin.New()

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
