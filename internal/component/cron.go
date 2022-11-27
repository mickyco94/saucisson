package component

import (
	"github.com/robfig/cron/v3"
)

func NewCronCondition(
	schedule string,
	cron *cron.Cron) *CronCondition {
	return &CronCondition{
		schedule: schedule,
		cron:     cron,
	}
}

type CronCondition struct {
	schedule string

	cron *cron.Cron
}

func (condition *CronCondition) Register(f func()) error {
	_, err := condition.cron.AddFunc(condition.schedule, f)
	return err
}
