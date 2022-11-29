package executor

import (
	"context"
	"sync"
)

type Pool struct {
	runningMu sync.Mutex
	wg        sync.WaitGroup
	ctx       context.Context

	size int
	//Internal queue for work
	//TODO: Send more context :)
	jobs chan Executor
}

func NewExecutorPool(context context.Context, size int) *Pool {
	return &Pool{
		ctx:       context,
		size:      size,
		wg:        sync.WaitGroup{},
		runningMu: sync.Mutex{},
		jobs:      make(chan Executor),
	}
}

func (pool *Pool) Stop() {
	close(pool.jobs)
	pool.wg.Wait()
}

func (pool *Pool) Run() {
	pool.wg.Add(pool.size)

	for i := 0; i < pool.size; i++ {
		go func() {
			defer pool.wg.Done()

			select {
			case <-pool.ctx.Done():
				return
			case j, open := <-pool.jobs:
				if !open {
					return
				}
				j.Execute()
			}
		}()
	}
}

// Enqueue adds the execution to the queue
func (pool *Pool) Enqueue(xc Executor) {
	pool.jobs <- xc
}
