package component

import (
	"gopkg.in/yaml.v3"
)

func (c *CronCondition) Configure(config yaml.Node) {
	cfg := &CronConfig{}
	config.Decode(cfg)

	c.Schedule = cfg.Schedule
}

type CronCondition struct {
	Schedule string
}

type CronConfig struct {
	Schedule string `yaml:"schedule"`
}
