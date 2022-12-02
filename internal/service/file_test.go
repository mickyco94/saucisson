package service

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/mickyco94/saucisson/internal/config"
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
	listener := NewFile(context.Background(), logrus.New())

	done := make(chan struct{})

	condition := &config.File{
		Path:      basePath,
		Operation: config.Create,
		Recursive: false,
	}

	err = listener.HandleFunc(condition, func() {
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

	listener := NewFile(context.Background(), logrus.New())

	done := make(chan struct{})

	condition := &config.File{
		Path:      basePath,
		Operation: config.Rename,
		Recursive: false,
	}

	err = listener.HandleFunc(condition, func() {
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

	listener := NewFile(context.Background(), logrus.New())

	condition := &config.File{
		Path:      filePath,
		Operation: config.Create,
		Recursive: false,
	}

	err = listener.HandleFunc(condition, func() {})

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

	listener := NewFile(context.Background(), logrus.New())

	done := make(chan struct{})

	condition := &config.File{
		Path:      basePath,
		Operation: config.Remove,
		Recursive: false,
	}

	err = listener.HandleFunc(condition, func() {
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

func TestMultipleWatchersForSomeFile(t *testing.T) {
	//arrange
	basePath, err := setup()

	if err != nil {
		t.Error(err)
	}

	listener := NewFile(context.Background(), logrus.New())

	handlerOne := make(chan struct{})
	handlerTwo := make(chan struct{})
	agg := mergeChans(handlerOne, handlerTwo)

	condition := &config.File{
		Path:      basePath,
		Operation: config.Create,
		Recursive: false,
	}

	err = listener.HandleFunc(condition, func() {
		handlerOne <- struct{}{}
	})

	err = listener.HandleFunc(condition, func() {
		handlerTwo <- struct{}{}
	})

	if err != nil {
		t.Error(err)
	}

	listener.Run(time.Millisecond * 100)

	//act
	err = ioutil.WriteFile(path.Join(basePath, "create.txt"), []byte("foo_bar"), 0644)
	if err != nil {
		t.Error(err)
	}

	//assert
	select {
	case <-time.NewTimer(1 * time.Second).C:
		t.Error("Timed out")
	case <-agg:
	}
}

func mergeChans(cs ...<-chan struct{}) <-chan struct{} {
	wg := sync.WaitGroup{}
	agg := make(chan struct{})

	wg.Add(len(cs))
	for _, c := range cs {
		go func(c <-chan struct{}) {
			for v := range c {
				agg <- v
			}
			wg.Done()
		}(c)
	}
	go func() {
		wg.Wait()
		close(agg)
	}()
	return agg
}
