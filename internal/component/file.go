package component

import (
	"errors"

	filewatcher "github.com/radovskyb/watcher"
	"gopkg.in/yaml.v3"
)

func (file *File) Configure(yaml yaml.Node) error {
	fileconfig := &FileConfig{}
	yaml.Decode(fileconfig)

	switch fileconfig.Operation {
	case "create":
		file.Operation = filewatcher.Create
	case "update":
		file.Operation = filewatcher.Write
	case "remove":
		file.Operation = filewatcher.Remove
	case "rename":
		file.Operation = filewatcher.Rename
	default:
		return errors.New("Unsupported operation")
	}
	file.Path = fileconfig.Path
	file.Recursive = fileconfig.Recursive
	return nil
}

// Possibly split between file + folder...?
// Both actually use FileListener but have more sensible config options
type File struct {
	Path      string
	Operation filewatcher.Op
	Recursive bool
}

type FileConfig struct {
	Operation string `yaml:"operation"`
	Path      string `yaml:"path"`
	Recursive bool   `yaml:"recursive"`
}
