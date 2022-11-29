package component

import (
	"github.com/robfig/cron/v3"
	"gopkg.in/yaml.v3"
)

func NewCron(
	cron *cron.Cron) *CronCondition {
	return &CronCondition{
		cron: cron,
	}
}

func (c *CronCondition) Configure(config yaml.Node) {
	cfg := &CronConfig{}
	config.Decode(cfg)

	c.schedule = cfg.Schedule
}

type CronCondition struct {
	schedule string

	cron *cron.Cron
}

func (condition *CronCondition) Register(f func()) error {
	_, err := condition.cron.AddFunc(condition.schedule, f)
	return err
}

type CronConfig struct {
	Schedule string `yaml:"schedule"`
}
