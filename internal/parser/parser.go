package parser

import (
	"io"

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
		Condition []ComponentSpec
		Execute   []ComponentSpec
	}
}

type ComponentSpec struct {
	Type   string    `yaml:"type"`
	Config yaml.Node `yaml:"config"`
}

func (r *RawConfig) Parse(raw io.Reader) error {

	decoder := yaml.NewDecoder(raw)
	err := decoder.Decode(r)

	if err != nil {
		return err
	}

	return nil
}
