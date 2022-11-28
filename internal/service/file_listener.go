package service

import (
	"context"
	"log"
	"time"

	filewatcher "github.com/radovskyb/watcher"
)

func NewFileListener(ctx context.Context) *FileListener {
	return &FileListener{
		context: ctx,
		Watcher: filewatcher.New(),
		Entries: make(map[string]*FileEntry),
	}
}

type FileEntry struct {
	file string
	op   filewatcher.Op
	h    func()
}

type FileListener struct {
	context context.Context
	Watcher *filewatcher.Watcher
	Entries map[string]*FileEntry
}

func (fl *FileListener) Stop() {
	fl.Watcher.Close()
}

func (fl *FileListener) AddFunc(op filewatcher.Op, path string, f func()) error {
	err := fl.Watcher.Add(path)
	if err != nil {
		return err
	}
	//Path is not a very unique identifier, you could have multiple for one path
	fl.Entries[path] = &FileEntry{
		file: path,
		op:   op,
		h:    f,
	}

	return nil
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

				entry := f.Entries[event.Path]
				if entry != nil && event.Op == entry.op {
					entry.h()
				}
			case err, ok := <-f.Watcher.Error:
				if !ok {
					return
				}
				log.Printf("error: %v\n", err)
			}
		}
	}()
}
