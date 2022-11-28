package component

import (
	"errors"

	"github.com/mickyco94/saucisson/internal/service"
	filewatcher "github.com/radovskyb/watcher"
)

func (fileConfig *FileConfig) FromConfig(listener *service.FileListener) (*File, error) {
	var op filewatcher.Op
	switch fileConfig.Operation {
	case "create":
		op = filewatcher.Create
	case "update":
		op = filewatcher.Write
	case "remove":
		op = filewatcher.Remove
	case "rename":
		op = filewatcher.Rename
	default:
		return nil, errors.New("Unsupported operation")
	}

	return NewFile(fileConfig.Path, op, fileConfig.Recursive, listener), nil
}

func NewFile(
	path string,
	op filewatcher.Op,
	recursive bool,
	listener *service.FileListener) *File {
	return &File{
		path:      path,
		op:        op,
		recursive: recursive,
		parent:    listener,
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
