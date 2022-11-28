package service

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/radovskyb/watcher"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func setup() (string, error) {
	path, err := ioutil.TempDir("", "file_listener_test")
	if err != nil {
		return "", err
	}
	return path, nil
}

func TestWatchDirectoryForCreateOp(t *testing.T) {
	basePath, err := setup()
	if err != nil {
		t.Error(err)
	}
	listener := NewFileListener(context.Background(), logrus.New())

	done := make(chan struct{})

	err = listener.AddFunc(watcher.Create, basePath, false, func() {
		done <- struct{}{}
	})

	if err != nil {
		t.Error(err)
	}

	listener.Run(time.Millisecond * 100)

	err = ioutil.WriteFile(path.Join(basePath, "create.txt"), []byte("foo_bar"), 0644)

	if err != nil {
		t.Error(err)
	}

	select {
	case <-time.NewTimer(1 * time.Second).C:
		t.Error("Timed out")
	case <-done:
		t.Logf("Handler invoked")
	}
}

func TestWatchForRename(t *testing.T) {
	//arrange
	basePath, err := setup()

	if err != nil {
		t.Error(err)
	}

	originalPath := path.Join(basePath, "rename.txt")
	err = ioutil.WriteFile(originalPath, []byte("foo_bar"), 0644)

	listener := NewFileListener(context.Background(), logrus.New())

	done := make(chan struct{})

	err = listener.AddFunc(watcher.Rename, basePath, false, func() {
		done <- struct{}{}
	})

	listener.Run(time.Millisecond * 100)

	//act
	newPath := path.Join(basePath, "rename_new.txt")

	err = os.Rename(originalPath, newPath)
	if err != nil {
		t.Error(err)
	}

	//assert
	select {
	case <-time.NewTimer(1 * time.Second).C:
		t.Error("Timed out")
	case <-done:

	}
}

func TestWatchCreateForExistingFileReturnsError(t *testing.T) {
	//arrange
	basePath, err := setup()

	if err != nil {
		t.Error(err)
	}

	filePath := path.Join(basePath, "exists.txt")
	err = ioutil.WriteFile(filePath, []byte("foo_bar"), 0644)

	listener := NewFileListener(context.Background(), logrus.New())

	err = listener.AddFunc(watcher.Create, filePath, false, func() {})

	assert.Error(t, err, ErrWatchCreateExistingFile)
	assert.Len(t, listener.entries, 0)
	assert.Len(t, listener.watcher.WatchedFiles(), 0)
}

func TestWatchFileRemoval(t *testing.T) {
	//arrange
	basePath, err := setup()

	if err != nil {
		t.Error(err)
	}

	filePath := path.Join(basePath, "delete_me.txt")
	err = ioutil.WriteFile(filePath, []byte("foo_bar"), 0644)

	listener := NewFileListener(context.Background(), logrus.New())

	done := make(chan struct{})

	err = listener.AddFunc(watcher.Remove, basePath, false, func() {
		done <- struct{}{}
	})

	if err != nil {
		t.Error(err)
	}

	listener.Run(time.Millisecond * 100)

	//act
	err = os.Remove(filePath)
	if err != nil {
		t.Error(err)
	}

	//assert
	select {
	case <-time.NewTimer(1 * time.Second).C:
		t.Error("Timed out")
	case <-done:
	}
}
