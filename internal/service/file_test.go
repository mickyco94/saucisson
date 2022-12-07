package service

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/mickyco94/saucisson/internal/config"
	filewatcher "github.com/radovskyb/watcher"
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

func TestMatches(t *testing.T) {
	type testCase struct {
		Event   filewatcher.Event
		Entry   fileEntry
		Matches bool
	}

	testCases := []testCase{
		{
			Event: filewatcher.Event{
				Op:   filewatcher.Create,
				Path: "/home/file.txt",
			},
			Entry: fileEntry{
				path: "/home",
				dir:  true,
				op:   filewatcher.Create,
			},
			Matches: true,
		},
		{
			Event: filewatcher.Event{
				Op:   filewatcher.Create,
				Path: "/home/sub/file.txt",
			},
			Entry: fileEntry{
				path: "/home",
				dir:  true,
				op:   filewatcher.Create,
			},
			Matches: false,
		},
		{
			Event: filewatcher.Event{
				Op:      filewatcher.Rename,
				Path:    "/home/new.txt",
				OldPath: "/home/old.txt",
			},
			Entry: fileEntry{
				path: "/home/old.txt",
				dir:  true,
				op:   filewatcher.Rename,
			},
			Matches: true,
		},
	}

	for _, testCase := range testCases {
		result := testCase.Entry.matches(testCase.Event)
		assert.Equal(t, result, testCase.Matches)
	}
}
