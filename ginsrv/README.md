# ginsrv

`ginsrv` is a lightweight package that simplifies the setup of a Gin router by defining routes and applying middlewares efficiently.

## Installation

```sh
go get github.com/seb7887/gofw/ginsrv
```

## Usage

### Import the package

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/seb7887/gofw/ginsrv"
)
```

### Define Routes and Setup the Router

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/yourusername/ginsrv"
    "net/http"
)

func main() {
    routes := []ginsrv.Route{
        {
            Method:  "GET",
            Path:    "/ping",
            Handler: func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "pong"}) },
        },
    }

    router := ginsrv.SetupRouter(routes)
    router.Run(":8080") // Start the server on port 8080
}
```

### Adding Middlewares

```go
func LoggerMiddleware(c *gin.Context) {
    // Custom logging logic
    c.Next()
}

func main() {
    routes := []ginsrv.Route{
        {
            Method:  "GET",
            Path:    "/hello",
            Handler: func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "Hello, World!"}) },
        },
    }

    router := ginsrv.SetupRouter(routes, LoggerMiddleware)
    router.Run(":8080")
}
```

## Features

- Simplified route definitions.
- Easy middleware application.
- Uses `gin.New()` for a clean router instance.
- Middleware is applied in reverse order to maintain expected execution flow.

## License
MIT

