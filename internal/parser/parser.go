package parser

import (
	"os"

	"gopkg.in/yaml.v3"
)

// These can all move to their respective paths
// With defaults set
type FileConfig struct {
	Operation string `yaml:"operation"`
	Path      string `yaml:"path"`
}

type CronConfig struct {
	Schedule string `yaml:"schedule"`
}

type ShellConfig struct {
	Command string `yaml:"command"`
}

type RawConfig struct {
	Services []struct {
		Name      string
		Condition struct {
			File *FileConfig `yaml:"file,omitempty"`
			Cron *CronConfig `yaml:"cron,omitempty"`
		}
		Execute struct {
			Shell *ShellConfig `yaml:"shell,omitempty"`
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
