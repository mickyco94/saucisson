package condition

import (
	"gopkg.in/yaml.v3"
)

func (c *Cron) Configure(config yaml.Node) {
	cfg := &cronConfig{}
	config.Decode(cfg)

	c.Schedule = cfg.Schedule
}

type Cron struct {
	Schedule string
}

type cronConfig struct {
	Schedule string `yaml:"schedule"`
}
