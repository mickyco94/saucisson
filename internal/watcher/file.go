package watcher

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mickyco94/saucisson/internal/config"
	filewatcher "github.com/radovskyb/watcher"
	"github.com/sirupsen/logrus"
)

var (
	ErrFileWatcherAlreadyClosed = errors.New("Already stopped")
	ErrWatchCreateExistingFile  = errors.New("Cannot watch for creation of a file that already exists")
)
var operationMap = map[config.Operation]filewatcher.Op{
	config.Create: filewatcher.Create,
	config.Remove: filewatcher.Remove,
	config.Rename: filewatcher.Rename,
	config.Update: filewatcher.Write,
}

func NewFile(logger logrus.FieldLogger) *File {

	watcher := filewatcher.New()
	watcher.IgnoreHiddenFiles(false)

	return &File{
		runningMu: sync.Mutex{},
		isRunning: false,
		close:     make(chan struct{}),
		done:      make(chan struct{}),

		logger:  logger,
		watcher: watcher,
	}
}

type fileEntry struct {
	//path is the full path of the file/directory being watched
	path string
	//dir is set to true if the specified entry is a watch for a directory
	dir bool
	//op is the type of operations we are listening for
	op filewatcher.Op
	//handler will be executed when a match is found
	handler func()
}

type File struct {
	runningMu sync.Mutex
	isRunning bool
	close     chan struct{}
	done      chan struct{}

	logger logrus.FieldLogger

	entries []fileEntry
	watcher *filewatcher.Watcher
}

func (file *File) Stop(ctx context.Context) error {
	file.runningMu.Lock()
	defer file.runningMu.Unlock()

	if !file.isRunning {
		return ErrFileWatcherAlreadyClosed
	}

	file.close <- struct{}{}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-file.done:
		return nil
	}
}

// HandleFunc registers the provided function to be executed, when the provided
// condition has been satisfied.
// An error is returned if the provided condition is not logically complete
func (f *File) HandleFunc(fileCondition *config.File, observer func()) error {

	file, err := os.Stat(fileCondition.Path)

	if file != nil &&
		!file.IsDir() &&
		operationMap[fileCondition.Operation] == filewatcher.Create {
		return ErrWatchCreateExistingFile
	}

	err = f.watcher.Add(fileCondition.Path)

	if err != nil {
		return err
	}

	f.entries = append(f.entries, fileEntry{
		path:    fileCondition.Path,
		dir:     file.IsDir(),
		op:      operationMap[fileCondition.Operation],
		handler: observer,
	})

	return nil
}

func (entry fileEntry) matches(event filewatcher.Event) bool {
	if event.Op != entry.op {
		return false
	}

	if event.Path == entry.path {
		return true
	}

	if entry.op == filewatcher.Rename && event.OldPath == entry.path {
		return true
	}

	if entry.dir && entry.path == filepath.Dir(event.Path) {
		return true
	}

	return false
}

func (file *File) Run(pollingInterval time.Duration) error {
	file.runningMu.Lock()

	if file.isRunning {
		file.runningMu.Unlock()
		return errors.New("Already running")
	}

	go func() {
		defer func() {
			file.done <- struct{}{}
		}()

		for {
			select {
			case <-file.close:
				return
			case event, open := <-file.watcher.Event:
				if !open {
					return
				}

				for _, entry := range file.entries {
					if entry.matches(event) {
						entry.handler()
					}
				}

			case err := <-file.watcher.Error:
				log.Printf("error: %v\n", err)
			}
		}
	}()

	file.isRunning = true
	file.runningMu.Unlock()

	return file.watcher.Start(pollingInterval)
}
