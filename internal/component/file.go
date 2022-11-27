package component

import (
	"github.com/mickyco94/saucisson/internal/service"
	filewatcher "github.com/radovskyb/watcher"
)

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
