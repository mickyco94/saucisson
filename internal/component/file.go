package component

import (
	"errors"
	"fmt"

	"github.com/mickyco94/saucisson/internal/dependencies"
	"github.com/mickyco94/saucisson/internal/parser"
	filewatcher "github.com/radovskyb/watcher"
)

//File is a condition that is triggered when a file
//is created, or if a directory is specified
//Then when a file is created in that directory

func NewFile(
	path string,
	op filewatcher.Op,
	listener *dependencies.FileListener) *File {
	return &File{
		path:   path,
		op:     op,
		parent: listener,
	}
}

// TODO: Propagate the file path as a param and allow it to be used in executors dynamically
type File struct {
	path   string
	op     filewatcher.Op
	parent *dependencies.FileListener
}

func (file *File) getRealPath() (string, error) {

	//File or directory exists, no problems
	return file.path, nil
}

func (file *File) Register(f func()) error {

	err := file.parent.Watcher.Add(file.path)
	if err != nil {
		return err
	}
	//Path is not a very unique identifier, you could have multiple for one path
	file.parent.Entries[file.path] = &dependencies.FileEntry{
		File: file.path,
		Dir:  file.path,
		Op:   file.op,
		H:    f,
	}
	return nil
}

func FileListenerFactory(c parser.Raw, deps *dependencies.Dependencies) (Condition, error) {

	path, err := c.ExtractString("path")
	if err != nil {
		return nil, err
	}

	op, err := c.ExtractString("operation")

	if err != nil {
		return nil, err
	}

	var actual filewatcher.Op
	switch op {
	case "create":
		actual = filewatcher.Create
	case "rename":
		actual = filewatcher.Rename
	case "delete":
		actual = filewatcher.Remove
	case "chmod":
		actual = filewatcher.Chmod
	case "update":
		actual = filewatcher.Write
	default:
		return nil, errors.New("Unsupported op")
	}

	fmt.Printf("Deps: %v\n", deps)

	return NewFile(path, actual, deps.FileListener), nil
}
