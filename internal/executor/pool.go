package executor

import (
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
// by `executor.Execute`
type Pool struct {
	runningMu sync.Mutex
	running   bool
	size      int

	wg     sync.WaitGroup
	logger logrus.FieldLogger

	jobs chan Job
}

func NewExecutorPool(logger logrus.FieldLogger, size int) *Pool {
	return &Pool{
		size:      size,
		wg:        sync.WaitGroup{},
		running:   false,
		runningMu: sync.Mutex{},
		logger:    logger,
		jobs:      make(chan Job),
	}
}

func (pool *Pool) Stop() {
	pool.runningMu.Lock()
	defer pool.runningMu.Unlock()

	if !pool.running {
		return
	}

	close(pool.jobs)
	pool.wg.Wait()
	pool.running = false
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
			defer pool.wg.Done()

			for j := range pool.jobs {
				//TODO: Error handle
				err := j.Executor.Execute()
				if err != nil {
					pool.logger.
						WithError(err).
						WithField("svc", j.Service).
						Error("Execution failed")
				}
			}
		}()
	}
}

// Enqueue adds the execution to the queue
func (pool *Pool) Enqueue(j Job) {
	pool.jobs <- j
}
