package service

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	filewatcher "github.com/radovskyb/watcher"
	"github.com/sirupsen/logrus"
)

var (
	ErrWatchCreateExistingFile = errors.New("error msg goes here")
)

func NewFileListener(
	ctx context.Context,
	logger logrus.FieldLogger) *FileListener {

	watcher := filewatcher.New()
	watcher.IgnoreHiddenFiles(false) //Decide this on a case by case basis

	return &FileListener{
		context: ctx,
		logger:  logger,
		watcher: filewatcher.New(),
		entries: make([]fileEntry, 0),
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

	watcher *filewatcher.Watcher
	entries []fileEntry
}

func (fl *FileListener) Stop() {
	fl.watcher.Close()
	close(fl.watcher.Event)
}

// Change to be a func (watcher.Event)
// Entry can then just go back to being paths + funcs
// Need to be careful that all matching
// ! Could just have all funcs invoked for every event, condition is always checked...?
func (fl *FileListener) AddFunc(op filewatcher.Op, path string, recursive bool, f func()) error {

	var entry fileEntry
	fileInfo, err := os.Stat(path)

	if err == os.ErrNotExist {

		//Try and get the dir
		dir := filepath.Dir(path)
		_, err := os.Open(dir)
		if err != nil {
			return err
		}

		entry = fileEntry{
			path: dir,
			op:   op,
			h:    f,
		}
	} else if err != nil {
		return err
	} else if op == filewatcher.Create && !fileInfo.IsDir() {
		return ErrWatchCreateExistingFile
	}

	entry = fileEntry{
		path:      path,
		op:        op,
		h:         f,
		recursive: recursive && fileInfo.IsDir(), //Cannot recursively watch a file
	}

	fl.entries = append(fl.entries, entry)

	if entry.recursive {
		return fl.watcher.AddRecursive(entry.path)
	} else {
		return fl.watcher.Add(entry.path)
	}
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

				f.logger.
					WithField("event", event).
					Debug("Event")

				for _, v := range f.entries {
					if v.matches(event) {
						v.h()
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
