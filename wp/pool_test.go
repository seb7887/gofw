package wp

import (
	"sync"
	"testing"
	"time"
)

func TestWorkerPool_BasicExecution(t *testing.T) {
	p := NewPool(3, 10)
	defer p.Stop()

	var (
		mu      sync.Mutex
		results []int
	)

	tasks := []struct {
		uid  string
		exec func()
	}{
		{"task1", func() { mu.Lock(); results = append(results, 1); mu.Unlock() }},
		{"task2", func() { mu.Lock(); results = append(results, 2); mu.Unlock() }},
		{"task3", func() { mu.Lock(); results = append(results, 3); mu.Unlock() }},
	}

	for _, task := range tasks {
		p.Submit(task.uid, task.exec)
	}

	time.Sleep(time.Millisecond * 500)

	if len(results) != 3 {
		t.Errorf("Wrong number of results: %d", len(results))
	}
}

func TestWorkerPool_GracefullyShutdown(t *testing.T) {
	p := NewPool(3, 5)
	var (
		counter int
		mu      sync.Mutex
	)

	tasks := 10
	for i := 0; i < tasks; i++ {
		p.Submit("task", func() {
			mu.Lock()
			counter++
			mu.Unlock()
		})
	}

	p.Stop()

	if counter != tasks {
		t.Errorf("Wrong number of tasks: %d", counter)
	}
}

func TestWorkerPool_NoPanicOnNilTask(t *testing.T) {
	p := NewPool(2, 2)
	defer p.Stop()

	p.Submit("nil-task", nil)

	time.Sleep(time.Millisecond * 100)
}

func TestWorkerPool_NoSubmitAfterStop(t *testing.T) {
	p := NewPool(2, 2)
	p.Stop()

	p.Submit("stopped-task", func() {})
}
