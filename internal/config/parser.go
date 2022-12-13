package config

import (
	"io"

	"gopkg.in/yaml.v3"
)

// Raw is the unprocessed configuration specification for the saucisson service
type Raw struct {
	Services []ServiceSpec `yaml:"services"`
}

// ServiceSpec is a structural definition of a service configuration,
// mirroring exactly how it is defined in YAML
type ServiceSpec struct {
	Name      string        `yaml:"name"`
	Condition ComponentSpec `yaml:"condition"`
	Execute   ComponentSpec `yaml:"execute"`
}

// ComponentSpec is a generic struct that corresponds
// to a condition or executor element in the YAML specification
type ComponentSpec struct {
	Type   Condition `yaml:"type"`
	Config yaml.Node `yaml:"config"`
}

// Parse reads config from the specified reader into the struct
func (r *Raw) Parse(reader io.Reader) error {

	decoder := yaml.NewDecoder(reader)
	err := decoder.Decode(r)

	if err != nil {
		return err
	}

	return nil
}
