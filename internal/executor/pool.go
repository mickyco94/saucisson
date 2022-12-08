package executor

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
)

// Job represents a unit of work
// triggered by a Service definitions condition(s) being satisfied
type Job struct {
	Service  string
	Executor Executor
}

// Pool represents a collection of workers that can be used
// by `executor.Execute` to dispatch work
type Pool struct {
	//ctx can be cancelled when `Stop()` is called.
	//This propagates to all running Executors which we wait to finish
	ctx    context.Context
	cancel context.CancelFunc

	runningMu sync.Mutex
	running   bool
	size      int

	wg     sync.WaitGroup
	logger logrus.FieldLogger

	jobs chan Job
}

var DefaultPoolSize = 15

func NewExecutorPool(logger logrus.FieldLogger) *Pool {
	localCtx, cancel := context.WithCancel(context.Background())

	return &Pool{
		ctx:    localCtx,
		cancel: cancel,

		size:      DefaultPoolSize,
		wg:        sync.WaitGroup{},
		running:   false,
		runningMu: sync.Mutex{},
		logger:    logger,
		jobs:      make(chan Job),
	}
}

func (pool *Pool) Stop(ctx context.Context) error {
	pool.runningMu.Lock()
	defer pool.runningMu.Unlock()

	if !pool.running {
		return nil
	}

	pool.running = false
	wgchan := make(chan bool)

	go func() {
		pool.wg.Wait()
		wgchan <- true
	}()

	//Stop sending new jobs
	close(pool.jobs)
	//Cancel all running jobs
	pool.cancel()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-wgchan:
		return nil
	}
}

func (pool *Pool) Run() {
	pool.runningMu.Lock()
	defer pool.runningMu.Unlock()
	if pool.running {
		return
	}
	pool.running = true

	pool.wg.Add(pool.size)

	for i := 0; i < pool.size; i++ {
		go func() {
			defer func() {
				pool.wg.Done()
			}()

			for job := range pool.jobs {
				err := job.Executor.Execute(pool.ctx)
				if err != nil {
					pool.logger.
						WithError(err).
						WithField("svc", job.Service).
						Error("Execution failed")
				}
			}
		}()
	}
}

// Enqueue adds the execution to the queue
func (pool *Pool) Enqueue(job Job) {
	pool.jobs <- job
}
