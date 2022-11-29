package service

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mickyco94/saucisson/internal/condition"
	filewatcher "github.com/radovskyb/watcher"
	"github.com/sirupsen/logrus"
)

var (
	ErrWatchCreateExistingFile = errors.New("Cannot watch for creation of a file that already exists")
)

func NewFileListener(
	ctx context.Context,
	logger logrus.FieldLogger) *FileListener {

	watcher := filewatcher.New()
	watcher.IgnoreHiddenFiles(false) //Decide this on a case by case basis

	return &FileListener{
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

type FileListener struct {
	context context.Context
	logger  logrus.FieldLogger

	entries []fileEntry
	watcher *filewatcher.Watcher
}

func (fl *FileListener) Stop() {
	fl.watcher.Close()
	close(fl.watcher.Event)
}

func (f *FileListener) HandleFunc(fileCondition *condition.File, observer func()) error {

	file, err := os.Stat(fileCondition.Path)

	if file != nil && !file.IsDir() && fileCondition.Operation == filewatcher.Create {
		return ErrWatchCreateExistingFile
	}

	err = f.watcher.Add(fileCondition.Path)

	f.entries = append(f.entries, fileEntry{
		path:      fileCondition.Path,
		op:        fileCondition.Operation,
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

func (f *FileListener) Run(pollingInterval time.Duration) {

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
