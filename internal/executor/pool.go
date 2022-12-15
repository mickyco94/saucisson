package executor

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
)

// Job represents a unit of work to be run by the executor pool
// triggered by a Service definitions condition(s) being satisfied
// Jobs can be run by adding them to the pool backlog via:
//
//	pool.Enqueue(job)
type Job struct {
	Service  string
	Executor ExecutorFunc
}

// Pool represents a collection of workers that can be used
// by `executor.Execute` to dispatch work
type Pool struct {
	logger logrus.FieldLogger
	ctx    context.Context
	cancel context.CancelFunc

	runningMu sync.Mutex
	running   bool

	size int
	wg   sync.WaitGroup
	jobs chan Job
}

// DefaultPoolSize represents the total number of goroutines
// that this executor pool shares. In the future this may be configurable
// and/or dependent on the number of services specified in configuration
var DefaultPoolSize = 15

// NewPool constructs a new executor pool
func NewPool(logger logrus.FieldLogger, size int) *Pool {
	localCtx, cancel := context.WithCancel(context.Background())

	return &Pool{
		ctx:       localCtx,
		cancel:    cancel,
		size:      size,
		wg:        sync.WaitGroup{},
		running:   false,
		runningMu: sync.Mutex{},
		logger:    logger,
		jobs:      make(chan Job),
	}
}

// Stop closes all running goroutines that are members of the
// execution pool, each executor is also instructured to cancel
// execution by an internal context.
//
// The context passed to the Stop method can be used to abort the shutdown of
// the executor pool
// e.g.
//
//	ctx, _ := context.WithTimeout(context.Background(), time.Second)
//	err := pool.Stop(ctx)
func (pool *Pool) Stop(ctx context.Context) error {
	pool.runningMu.Lock()

	if !pool.running {
		pool.runningMu.Unlock()
		return nil
	}

	pool.running = false
	pool.runningMu.Unlock()

	runningContext, cancel := context.WithCancel(context.Background())

	go func() {
		pool.wg.Wait()
		cancel()
	}()

	//Stop sending new jobs
	close(pool.jobs)
	//Cancel all running jobs
	pool.cancel()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-runningContext.Done():
		return nil
	}
}

// Start spawns N executors in the pool.
// If the pool is already running then the operations is a no-op
//
// Closing of the pool is completed using:
//
//	pool.Stop(context.Context)
func (pool *Pool) Start() {
	pool.runningMu.Lock()
	if pool.running {
		pool.runningMu.Unlock()
		return
	}

	pool.running = true
	pool.runningMu.Unlock()

	pool.wg.Add(pool.size)

	for i := 0; i < pool.size; i++ {
		go func() {
			defer func() {
				pool.wg.Done()
			}()

			for job := range pool.jobs {
				pool.run(&job)
			}
		}()
	}
}

// Enqueue adds the execution to the queue
func (pool *Pool) Enqueue(job Job) {
	pool.jobs <- job
}

// run executes a job on the pool and manages all error handling
func (pool *Pool) run(job *Job) {
	defer func() {
		if rec := recover(); rec != nil {
			pool.logger.
				WithField("svc", job.Service).
				WithField("panic", rec).
				Error("Executor panicked")
		}
	}()

	err := job.Executor(pool.ctx)
	if err != nil {
		pool.logger.
			WithError(err).
			WithField("svc", job.Service).
			Error("Execution failed")
	} else {
		pool.logger.
			WithField("svc", job.Service).
			Info("Execution completed")
	}
}
