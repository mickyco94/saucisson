package service

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/mickyco94/saucisson/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func setup() string {
	path, err := ioutil.TempDir("", "file_listener_test")
	if err != nil {
		panic(err)
	}
	return path
}

func createDummyFile(base string) string {
	path := path.Join(base, "dummy.txt")
	err := ioutil.WriteFile(path, []byte("foo_bar"), 0644)

	if err != nil {
		panic(err)
	}

	return path
}

func TestDirectory(t *testing.T) {
	basePath := setup()
	listener := NewFile(logrus.New())

	done := make(chan struct{})

	condition := &config.File{
		Path:      basePath,
		Operation: config.Create,
		Recursive: false,
	}

	listener.HandleFunc(condition, func() {
		done <- struct{}{}
	})

	go listener.Run(time.Millisecond * 100)

	createDummyFile(basePath)

	select {
	case <-time.After(time.Second):
		t.Error("Timed out")
	case <-done:
	}
}

func TestRename(t *testing.T) {
	basePath := setup()

	filePath := createDummyFile(basePath)

	listener := NewFile(logrus.New())

	done := make(chan struct{})

	condition := &config.File{
		Path:      basePath,
		Operation: config.Rename,
		Recursive: false,
	}

	listener.HandleFunc(condition, func() {
		done <- struct{}{}
	})

	go listener.Run(time.Millisecond * 100)

	newPath := path.Join(basePath, "rename.txt")

	os.Rename(filePath, newPath)

	select {
	case <-time.After(time.Second):
		t.Error("Timed out")
	case <-done:
	}
}

func TestFileAlreadyExists(t *testing.T) {
	//arrange
	basePath := setup()

	filePath := createDummyFile(basePath)

	listener := NewFile(logrus.New())

	condition := &config.File{
		Path:      filePath,
		Operation: config.Create,
		Recursive: false,
	}

	err := listener.HandleFunc(condition, func() {})

	assert.Error(t, err, ErrWatchCreateExistingFile)
}

func TestRemoval(t *testing.T) {
	basePath := setup()

	filePath := createDummyFile(basePath)

	listener := NewFile(logrus.New())

	done := make(chan struct{})

	condition := &config.File{
		Path:      basePath,
		Operation: config.Remove,
		Recursive: false,
	}

	listener.HandleFunc(condition, func() {
		done <- struct{}{}
	})

	go listener.Run(time.Millisecond * 100)

	err := os.Remove(filePath)

	if err != nil {
		t.Error(err)
	}

	select {
	case <-time.After(5 * time.Second):
		t.Error("Timed out")
	case <-done:
	}
}

func TestMultipleWatchersForSameFile(t *testing.T) {
	//arrange
	basePath := setup()

	listener := NewFile(logrus.New())

	one := make(chan struct{})
	two := make(chan struct{})

	condition := &config.File{
		Path:      basePath,
		Operation: config.Create,
		Recursive: false,
	}

	listener.HandleFunc(condition, func() {
		one <- struct{}{}
	})

	listener.HandleFunc(condition, func() {
		two <- struct{}{}
	})

	go listener.Run(time.Millisecond * 100)

	createDummyFile(basePath)

	select {
	case <-time.After(500 * time.Millisecond):
		t.Error("timeout")
	case <-one:
	}

	select {
	case <-time.After(500 * time.Millisecond):
		t.Error("timeout")
	case <-two:
	}
}
