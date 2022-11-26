package condition

import (
	"github.com/robfig/cron/v3"
)

func NewCronCondition(schedule string, cron *cron.Cron) *CronCondition {
	return &CronCondition{
		opts: Opts{
			schedule: schedule,
		},
		cron: cron,
	}
}

type Opts struct {
	schedule string
}

type CronCondition struct {
	opts Opts

	cron *cron.Cron
}

func (crn *CronCondition) Register(trigger chan<- struct{}) error {
	_, err := crn.cron.AddFunc(crn.opts.schedule, func() {
		trigger <- struct{}{}
	})
	return err
}
