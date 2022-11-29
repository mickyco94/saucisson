package service

import (
	"github.com/mickyco94/saucisson/internal/condition"
	internal "github.com/robfig/cron/v3"
)

type Cron struct {
	inner *internal.Cron
}

func NewCron() *Cron {
	return &Cron{
		inner: internal.New(internal.WithSeconds()),
	}
}

func (cron *Cron) HandleFunc(condition *condition.Cron, observer func()) {
	cron.inner.AddFunc(condition.Schedule, observer)
}

func (cron *Cron) Start() { cron.inner.Start() }

func (cron *Cron) Stop() { cron.inner.Stop() }
