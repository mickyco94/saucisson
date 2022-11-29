package parser

import (
	"io"

	"gopkg.in/yaml.v3"
)

// RawConfig
type RawConfig struct {
	Services []ServiceSpec
}

type ServiceSpec struct {
	Name      string        `yaml:"name"`
	Condition ComponentSpec `yaml:"condition"`
	Execute   ComponentSpec `yaml:"execute"`
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
