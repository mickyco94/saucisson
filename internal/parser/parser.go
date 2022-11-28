package parser

import (
	"os"

	"github.com/mickyco94/saucisson/internal/component"
	"gopkg.in/yaml.v3"
)

// These can all move to their respective paths
// With defaults set

type RawConfig struct {
	Services []struct {
		Name      string
		Condition struct {
			File *component.FileConfig `yaml:"file,omitempty"`
			Cron *component.CronConfig `yaml:"cron,omitempty"`
		}
		Execute struct {
			Shell *component.ShellConfig `yaml:"shell,omitempty"`
		}
	}
}

// TODO: This should be an io.Reader pattern probably
func Parse(path string) (*RawConfig, error) {
	rawConfig := &RawConfig{}

	bytes, err := os.ReadFile(path)

	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(bytes, rawConfig)

	if err != nil {
		return nil, err
	}

	return rawConfig, nil
}
