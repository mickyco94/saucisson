package condition

import "github.com/robfig/cron/v3"

func NewCronCondition(schedule string, cron *cron.Cron) *CronCondition {
	return &CronCondition{
		Schedule: schedule,
		cron:     cron,
	}
}

type CronCondition struct {
	Schedule string

	cron *cron.Cron
}

func (crn *CronCondition) Register(trigger chan<- struct{}) {
	crn.cron.AddFunc(crn.Schedule, func() {
		trigger <- struct{}{}
	})
}
