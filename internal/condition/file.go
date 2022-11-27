package condition

import (
	"context"
	"log"
	"time"

	filewatcher "github.com/radovskyb/watcher"
)

//File is a condition that is triggered when a file
//is created, or if a directory is specified
//Then when a file is created in that directory

func NewFile(
	path string,
	op filewatcher.Op,
	listener *FileListener) *File {
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
	parent *FileListener
}

func (file *File) getRealPath() (string, error) {

	//File or directory exists, no problems
	return file.path, nil
}

func (file *File) Register(f func()) error {

	err := file.parent.watcher.Add(file.path)
	if err != nil {
		return err
	}
	//Path is not a very unique identifier, you could have multiple for one path
	file.parent.entries[file.path] = &fileEntry{
		file: file.path,
		dir:  file.path,
		op:   file.op,
		h:    f,
	}
	return nil
}

type fileEntry struct {
	//The file we are watching for changes, nil if we are watching a dir
	file string
	//The directory we are watching
	dir string
	//The operation we are listening for
	op filewatcher.Op //h is executed if a match is found
	h  func()
}

type FileListener struct {
	context context.Context
	//Inner watcher
	watcher *filewatcher.Watcher

	entries map[string]*fileEntry //Multiple could apply...? map[string][]func()
	//Original paths that were watched and their corresponding functions
	//This needs to be more complex to allow
}

func (fl *FileListener) Stop() {
	fl.watcher.Close()
}

func NewFileListener(ctx context.Context) *FileListener {
	return &FileListener{
		context: ctx,
		watcher: filewatcher.New(),
		entries: make(map[string]*fileEntry),
	}
}

func (f *FileListener) Run() {

	//TODO: error trap
	go f.watcher.Start(100 * time.Millisecond)

	go func() {
		for {
			select {
			case <-f.context.Done():
				return
			case event, ok := <-f.watcher.Event:
				if !ok {
					return
				}

				log.Printf("Event: %v\n", event)

				entry := f.entries[event.Name()]
				if entry != nil && event.Op == entry.op {
					entry.h()
				}
			case err, ok := <-f.watcher.Error:
				if !ok {
					return
				}
				log.Println("error:", err)

			}
		}
	}()
}
