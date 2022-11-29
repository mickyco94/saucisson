package component

import (
	"errors"

	"github.com/mickyco94/saucisson/internal/service"
	filewatcher "github.com/radovskyb/watcher"
	"gopkg.in/yaml.v3"
)

func (file *File) Configure(yaml yaml.Node) error {
	fileconfig := &FileConfig{}
	yaml.Decode(fileconfig)

	switch fileconfig.Operation {
	case "create":
		file.op = filewatcher.Create
	case "update":
		file.op = filewatcher.Write
	case "remove":
		file.op = filewatcher.Remove
	case "rename":
		file.op = filewatcher.Rename
	default:
		return errors.New("Unsupported operation")
	}
	file.path = fileconfig.Path
	file.recursive = fileconfig.Recursive
	return nil
}

func NewFile(
	listener *service.FileListener) *File {
	return &File{
		parent: listener,
	}
}

// Possibly split between file + folder...?
// Both actually use FileListener but have more sensible config options
type File struct {
	path      string
	op        filewatcher.Op
	recursive bool

	parent *service.FileListener
}

func (file *File) Register(f func()) error {
	return file.parent.AddFunc(file.op, file.path, file.recursive, f)
}

type FileConfig struct {
	Operation string `yaml:"operation"`
	Path      string `yaml:"path"`
	Recursive bool   `yaml:"recursive"`
}
