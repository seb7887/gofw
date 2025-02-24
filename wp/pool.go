package wp

import (
	"context"
	"github.com/segmentio/fasthash/fnv1a"
	"sync"
)

type Pool struct {
	maxWorkers int
	taskQueues []chan func()
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewPool(maxWorkers int, queueBuffer int) *Pool {
	if maxWorkers < 1 {
		maxWorkers = 1
	}
	if queueBuffer < 1 {
		queueBuffer = 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	p := &Pool{
		maxWorkers: maxWorkers,
		taskQueues: make([]chan func(), maxWorkers),
		ctx:        ctx,
		cancel:     cancel,
	}

	for i := 0; i < maxWorkers; i++ {
		p.taskQueues[i] = make(chan func(), queueBuffer)
		p.wg.Add(1)
		go p.startWorker(p.taskQueues[i])
	}

	return p
}

func (p *Pool) startWorker(queue chan func()) {
	defer p.wg.Done()
	for {
		select {
		case task, ok := <-queue:
			if !ok {
				// closed channel
				return
			}
			task()
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *Pool) Submit(uid string, task func()) {
	if task == nil {
		return
	}
	idx := fnv1a.HashString64(uid) % uint64(p.maxWorkers)
	select {
	case p.taskQueues[idx] <- task:
	case <-p.ctx.Done():
	}
}

func (p *Pool) Stop() {
	p.cancel()
	for _, q := range p.taskQueues {
		close(q)
	}
	p.wg.Wait()
}
