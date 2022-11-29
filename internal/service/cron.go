package service

import (
	"github.com/mickyco94/saucisson/internal/component"
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

func (cron *Cron) AddCondition(condition component.CronCondition, observer func()) {
	cron.inner.AddFunc(condition.Schedule, observer)
}
