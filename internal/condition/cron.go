package condition

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

func (crn *CronCondition) Register(f func()) error {
	_, err := crn.cron.AddFunc(crn.schedule, f)
	return err
}
