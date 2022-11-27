package parser

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

//Define a generic config specification that has the concept of options
//We can then define constructors that take parsed config and return actual
//This can be our factory

//! There are several layers we need to build here:
//! - Generic definitions of components as structs, these are the specs
//! - Parsing of the yaml into something that can be queried. ! Start with this
//! - Taking the queryable struct and mapping to generic definitions
//! - Using a realised implementation of the generic definition to create an actual implementation
//! - What is the interface you would like/need from the consumer of parser?

//service.Register("service_name", conditionCtorFunc, executorCtorFunc)

//config.Parsed should be the relevant section...?
//resource refers to things like HttpClient, Logger, Cron etc.
//conditionCtorFunc := func(config *config.Parsed, resource *app.Resources) (condition.Condition, error) {
//
//}

//Define a map of type

type RawConfig struct {
	Services []struct {
		Name      string
		Condition Raw
		Execute   Raw
	}
}

// Can be avoided by being more strongly typed
func (r Raw) Name() (string, error) {
	i := 0
	key := ""
	for k := range r {
		if i > 0 {
			//TODO: Must be a better way than this lol
			return "", errors.New("Multiple keys specified for component declaration")
		}
		i++
		key = k
	}
	return key, nil
}

func (r Raw) Extract(key string) (any, error) {
	v, ok := r[key]
	if !ok {
		return nil, errors.New("Does not exist")
	}
	return v, nil
}

func (r Raw) ExtractString(key string) (string, error) {
	v, ok := r[key]
	if !ok {
		return "", errors.New("Key does not exist!")
	}
	return v.(string), nil
}

func (r Raw) ExtractSection(key string) (Raw, error) {
	v, ok := r[key]
	if !ok {
		return nil, errors.New("Section does not exist!")
	}
	return v.(Raw), nil
}

// ! This'll do for now
// Not the best interface to define but I can flesh this out
// When we come to defining a Linter
type Raw map[string]any

// TODO: This should be an io.Reader pattern probably
// parseConfig attempts to read a config from the specified path
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
