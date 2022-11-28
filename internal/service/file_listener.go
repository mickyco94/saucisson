package service

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
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
	return &FileListener{
		context: ctx,
		logger:  logger,
		watcher: filewatcher.New(),
		entries: make([]fileEntry, 0),
	}
}

type fileEntry struct {
	path string
	op   filewatcher.Op
	h    func()
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

func (fl *FileListener) AddFunc(op filewatcher.Op, path string, f func()) error {

	err := fl.watcher.Add(path)
	if err != nil {
		return err
	}

	fileInfo, err := os.Stat(path)

	if err == os.ErrNotExist {
		//Try and get the dir
		dir := filepath.Dir(path)
		_, err := os.Open(dir)
		if err != nil {
			return err
		}

		fl.entries = append(fl.entries, fileEntry{
			path: dir,
			op:   op,
			h:    f,
		})

		return nil
	}

	if op == filewatcher.Create && !fileInfo.IsDir() {
		return ErrWatchCreateExistingFile
	}

	fl.entries = append(fl.entries, fileEntry{
		path: path,
		op:   op,
		h:    f,
	})

	return nil
}

func (entry fileEntry) matches(event filewatcher.Event) bool {
	if event.Path == entry.path {
		return entry.op == event.Op
	}
	if entry.path == filepath.Dir(event.Path) {
		return entry.op == event.Op
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
