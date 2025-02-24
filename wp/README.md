# wp

A simple worker pool implementation in Go that allows executing tasks in parallel using a fixed number of workers.

## Installation

```bash
go get -u github.com/seb7887/gofw/wp
```

## Usage

```go
package main

import (
	"fmt"
	"time"

	"github.com/yourrepo/pool"
)

func main() {
	p := pool.New(3, 10) // 3 workers, task queue with buffer of 10
	defer p.Stop()

	for i := 0; i < 5; i++ {
		idx := i
		p.Submit(fmt.Sprintf("task-%d", i), func() {
			fmt.Printf("Executing task %d\n", idx)
			time.Sleep(1 * time.Second)
		})
	}

	time.Sleep(2 * time.Second) // Allow tasks to complete
}
```