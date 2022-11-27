package parser

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

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

func (r Raw) ExtractString(key string) (string, error) {
	v, ok := r[key]
	if !ok {
		return "", errors.New("Key does not exist!!!")
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
type Raw map[string]interface{}

type CronComponentConfiguration struct {
	Schedule string
}

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
