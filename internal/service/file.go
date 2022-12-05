package service

import (
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

func NewFile(logger logrus.FieldLogger) *File {

	watcher := filewatcher.New()
	watcher.IgnoreHiddenFiles(false) //Decide this on a case by case basis

	return &File{
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
	close chan struct{}
	done  chan struct{}

	logger logrus.FieldLogger

	entries []fileEntry
	watcher *filewatcher.Watcher
}

func (fl *File) Stop() {
	fl.watcher.Close()
	fl.close <- struct{}{}
	<-fl.done
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

func (f *File) Run(pollingInterval time.Duration) error {

	go func() {
		for {
			select {
			case <-f.close:
				f.logger.Debug("Shutting down file service")
				f.done <- struct{}{}
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

	return f.watcher.Start(pollingInterval)
}
