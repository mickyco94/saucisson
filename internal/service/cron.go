package service

import (
	"github.com/mickyco94/saucisson/internal/config"
	internal "github.com/robfig/cron/v3"
)

// Cron is a decorator of the cron lib
// this allows the `HandleFunc(config.Condition)` pattern to be established
// widely throughout the architecture
type Cron struct {
	inner *internal.Cron
}

func NewCron() *Cron {
	return &Cron{
		inner: internal.New(internal.WithSeconds()),
	}
}

func (cron *Cron) HandleFunc(condition *config.Cron, observer func()) {
	cron.inner.AddFunc(condition.Schedule, observer)
}

func (cron *Cron) Run() { cron.inner.Run() }

func (cron *Cron) Stop() { cron.inner.Stop() }
