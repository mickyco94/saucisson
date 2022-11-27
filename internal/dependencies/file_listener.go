package dependencies

import (
	"context"
	"log"
	"time"

	filewatcher "github.com/radovskyb/watcher"
)

type FileEntry struct {
	//The File we are watching for changes, nil if we are watching a dir
	File string
	//The directory we are watching
	Dir string
	//The operation we are listening for
	Op filewatcher.Op //h is executed if a match is found
	H  func()
}

type FileListener struct {
	context context.Context
	//Inner Watcher
	Watcher *filewatcher.Watcher

	Entries map[string]*FileEntry //Multiple could apply...? map[string][]func()
	//Original paths that were watched and their corresponding functions
	//This needs to be more complex to allow
}

func (fl *FileListener) Stop() {
	fl.Watcher.Close()
}

func NewFileListener(ctx context.Context) *FileListener {
	return &FileListener{
		context: ctx,
		Watcher: filewatcher.New(),
		Entries: make(map[string]*FileEntry),
	}
}

func (f *FileListener) Run() {

	//TODO: error trap
	go f.Watcher.Start(100 * time.Millisecond)

	go func() {
		for {
			select {
			case <-f.context.Done():
				return
			case event, ok := <-f.Watcher.Event:
				if !ok {
					return
				}

				log.Printf("Event: %v\n", event)

				entry := f.Entries[event.Name()]
				if entry != nil && event.Op == entry.Op {
					entry.H()
				}
			case err, ok := <-f.Watcher.Error:
				if !ok {
					return
				}
				log.Println("error:", err)

			}
		}
	}()
}
