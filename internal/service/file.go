package service

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mickyco94/saucisson/internal/config"
	filewatcher "github.com/radovskyb/watcher"
	"github.com/sirupsen/logrus"
)

var ErrWatchCreateExistingFile = errors.New("Cannot watch for creation of a file that already exists")

var operationMap = map[config.Operation]filewatcher.Op{
	config.Create: filewatcher.Create,
	config.Remove: filewatcher.Remove,
	config.Rename: filewatcher.Rename,
	config.Update: filewatcher.Write,
}

func NewFile(
	ctx context.Context,
	logger logrus.FieldLogger) *File {

	watcher := filewatcher.New()
	watcher.IgnoreHiddenFiles(false) //Decide this on a case by case basis

	return &File{
		context: ctx,
		logger:  logger,
		watcher: watcher,
	}
}

type fileEntry struct {
	path      string
	op        filewatcher.Op
	h         func()
	recursive bool
}

type File struct {
	context context.Context
	logger  logrus.FieldLogger

	entries []fileEntry
	watcher *filewatcher.Watcher
}

func (fl *File) Stop() {
	fl.watcher.Close()
	close(fl.watcher.Event)
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

	f.entries = append(f.entries, fileEntry{
		path:      fileCondition.Path,
		op:        operationMap[fileCondition.Operation],
		h:         observer,
		recursive: fileCondition.Recursive,
	})

	return err
}

func (entry fileEntry) matches(event filewatcher.Event) bool {
	if event.Op != entry.op {
		return false
	}

	if event.Path == entry.path {
		return true
	}

	//This is allowing reads one dir up
	if entry.path == filepath.Dir(event.Path) {
		return true
	}

	if entry.recursive && strings.Contains(event.Path, entry.path) {
		return true
	}

	return false
}

func (f *File) Run(pollingInterval time.Duration) {

	//We can safely ignore err here as the only cases are if
	//the watcher is already running or an invalid duration is set
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

				for _, entry := range f.entries {
					if entry.matches(event) {
						entry.h()
					}
				}

			case err, ok := <-f.watcher.Error:
				if !ok {
					return
				}
				log.Printf("error: %v\n", err)
			}
		}
	}()
}
