package condition

import (
	"context"
	"log"
	"strings"

	"github.com/fsnotify/fsnotify"
)

//File is a condition that is triggered when a file
//is created, or if a directory is specified
//Then when a file is created in that directory

func NewFile(path string, listener *FileListener) *File {
	return &File{
		path:   path,
		parent: listener,
	}
}

// TODO: Propagate the file path as a param and allow it to be used in executors dynamically
type File struct {
	path   string
	parent *FileListener
	op     fsnotify.Op
}

func (file *File) Register(f func()) error {
	//TODO: Wrap
	err := file.parent.watcher.Add(file.path)
	if err != nil {
		return err
	}
	//Path is not a very unique identifier, you could have multiple for one path
	file.parent.entries[file.path] = f
	return nil
}

type FileListener struct {
	context context.Context
	//Inner watcher
	watcher *fsnotify.Watcher

	entries map[string]func() //Original paths that were watched and their corresponding functions
	//This needs to be more complex to allow
}

func NewFileListener(ctx context.Context) *FileListener {
	watcher, err := fsnotify.NewWatcher() //!Ignore errors lol
	if err != nil {
		log.Printf("err: %v", err)
	}
	return &FileListener{
		context: ctx,
		watcher: watcher,
		entries: make(map[string]func()),
	}
}

func (f *FileListener) Run() {
	go func() {
		for {
			select {
			case <-f.context.Done():
				//More elegant
				return
			case event, ok := <-f.watcher.Events:
				if !ok {
					return
				}
				if !event.Has(fsnotify.Create) {
					continue
				}
				//! Inefficient, use a better datastructure:
				// The matching here needs to be more restrictive
				//There are special flags we can include, like "includeSubdirectories etc."
				//Different behaviours for files vs. folders. If a user listens for a file being created
				//That does not exist yet, we can just listen in that folder and then match for all the
				//Create events. There is some sophistication and config options that we can add as an extra
				//layer of this simply inotifyd wrapper
				//Determine what ~ means, etc.
				for k, v := range f.entries {

					if strings.Contains(event.Name, k) {
						v()
					}
				}
			case err, ok := <-f.watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)

			}
		}
	}()
}
