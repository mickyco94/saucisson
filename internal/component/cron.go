package component

import (
	"github.com/mickyco94/saucisson/internal/dependencies"
	"github.com/mickyco94/saucisson/internal/parser"
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

func CronFactory(c parser.Raw, r *dependencies.Dependencies) (Condition, error) {
	schedule, err := c.ExtractString("schedule")
	if err != nil {
		return nil, err
	}

	return NewCronCondition(schedule, r.Cron), nil
}
