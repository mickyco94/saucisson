package config

import (
	"io"

	"gopkg.in/yaml.v3"
)

type Raw struct {
	Services []ServiceSpec `yaml:"services"`
}

type ServiceSpec struct {
	Name      string        `yaml:"name"`
	Condition ComponentSpec `yaml:"condition"`
	Execute   ComponentSpec `yaml:"execute"`
}

type ComponentSpec struct {
	Type   Condition `yaml:"type"`
	Config yaml.Node `yaml:"config"`
}

func (r *Raw) Parse(raw io.Reader) error {

	decoder := yaml.NewDecoder(raw)
	err := decoder.Decode(r)

	if err != nil {
		return err
	}

	return nil
}
