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
	return NewFile(fileConfig.Path, op, listener), nil
}

func NewFile(
	path string,
	op filewatcher.Op,
	listener *service.FileListener) *File {
	return &File{
		path:   path,
		op:     op,
		parent: listener,
	}
}

type File struct {
	path   string
	op     filewatcher.Op
	parent *service.FileListener
}

func (file *File) Register(f func()) error {
	return file.parent.AddFunc(file.op, file.path, f)
}

type FileConfig struct {
	Operation string `yaml:"operation"`
	Path      string `yaml:"path"`
}
