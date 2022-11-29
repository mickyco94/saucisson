package parser

import (
	"io"

	"github.com/mickyco94/saucisson/internal/condition"
	"gopkg.in/yaml.v3"
)

// RawConfig
type RawConfig struct {
	Services []ServiceSpec
}

type ServiceSpec struct {
	Name string `yaml:"name"`
	//TODO: Take multiple conditions
	Condition ComponentSpec `yaml:"condition"`
	//TODO: Take multiple executors
	Execute ComponentSpec `yaml:"execute"`
}

type ComponentSpec struct {
	Type   condition.Condition `yaml:"type"`
	Config yaml.Node           `yaml:"config"`
}

func (r *RawConfig) Parse(raw io.Reader) error {

	decoder := yaml.NewDecoder(raw)
	err := decoder.Decode(r)

	if err != nil {
		return err
	}

	return nil
}
