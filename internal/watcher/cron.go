package watcher

import (
	"context"

	"github.com/mickyco94/saucisson/internal/config"
	internal "github.com/robfig/cron/v3"
)

// Cron is a decorator of the cron lib
// this allows the `HandleFunc(config.Condition)` pattern to be established
// widely throughout the architecture
type Cron struct {
	inner *internal.Cron
}

// NewCron constructs a new cron schedule watcher
func NewCron() *Cron {
	return &Cron{
		inner: internal.New(internal.WithSeconds()),
	}
}

// HandleFunc registers a function to be executed when the provided condition is met.
func (cron *Cron) HandleFunc(condition *config.Cron, handler func()) error {
	_, err := cron.inner.AddFunc(condition.Schedule, handler)
	return err
}

// Run starts the cron watcher on its own goroutine
func (cron *Cron) Run() { cron.inner.Run() }

// Stop shuts down the cron watcher and attempts to wait for any currently
// running functions attached to the scheduler to exit before the provided
// context is done.
func (cron *Cron) Stop(ctx context.Context) error {
	runningJobsCtx := cron.inner.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-runningJobsCtx.Done():
		return nil
	}
}
