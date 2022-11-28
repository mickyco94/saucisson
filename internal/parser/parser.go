package parser

import (
	"io"

	"github.com/mickyco94/saucisson/internal/component"
	"gopkg.in/yaml.v3"
)

// RawConfig
type RawConfig struct {
	Services []struct {

		//Name represents the friendly name for this service that can be
		//used to query the state of this service by the end user
		Name string `yaml:"name"`

		//Condition holds the configuration options for all conditions
		//of this service executing. If a condition is `nil` then it is ignored
		//as a requirement.
		Condition struct {
			File *component.FileConfig `yaml:"file,omitempty"`
			Cron *component.CronConfig `yaml:"cron,omitempty"`
		}

		Execute struct {
			Shell *component.ShellConfig `yaml:"shell,omitempty"`
		}
	}
}

func (r *RawConfig) Parse(raw io.Reader) error {

	decoder := yaml.NewDecoder(raw)
	err := decoder.Decode(r)

	if err != nil {
		return err
	}

	return nil
}
