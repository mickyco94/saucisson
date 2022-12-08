package watcher

import (
	"context"
	"testing"
	"time"

	"github.com/mickyco94/saucisson/internal/config"
	"github.com/mitchellh/go-ps"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// mockProcess mocks `ps.Process`
type mockProcess struct {
	executable string
}

func (m mockProcess) Executable() string {
	return m.executable
}

func (m mockProcess) PPid() int {
	return 1
}

func (m mockProcess) Pid() int {
	return 2
}

func (p *Process) setRunning(process string, isRunning bool) {

	if isRunning {
		p.source = func() ([]ps.Process, error) {
			return []ps.Process{mockProcess{executable: "top"}}, nil
		}
	} else {
		p.source = func() ([]ps.Process, error) {
			return []ps.Process{}, nil
		}
	}
}

func TestListenForOpen(t *testing.T) {

	called := make(chan struct{})

	proc := NewProcess(logrus.New())

	proc.HandleFunc(&config.Process{
		Executable: "top",
		State:      config.Open,
	}, func() {
		called <- struct{}{}
	})

	go proc.Run()

	<-time.After(1 * time.Second)

	proc.setRunning("top", true)

	select {
	case <-time.After(1 * time.Second):
		t.Fail()
	case <-called:
		t.Log("Passed")
	}

	proc.Stop(context.Background())
}

func TestCloseWhenAlreadyRunning(t *testing.T) {

	called := make(chan struct{})

	proc := NewProcess(logrus.New())

	proc.HandleFunc(&config.Process{
		Executable: "top",
		State:      config.Close,
	}, func() {
		called <- struct{}{}
	})

	proc.setRunning("top", true)

	go proc.Run()

	<-time.After(500 * time.Millisecond)

	proc.setRunning("top", false)

	select {
	case <-time.After(1 * time.Second):
		t.Fail()
	case <-called:
		t.Log("Passed")
	}

	proc.Stop(context.Background())
}

func TestOpenWhenAlreadyOpen(t *testing.T) {

	called := make(chan struct{})

	proc := NewProcess(logrus.New())

	proc.HandleFunc(&config.Process{
		Executable: "top",
		State:      config.Open,
	}, func() {
		called <- struct{}{}
	})

	proc.setRunning("top", true)

	go proc.Run()

	<-time.After(500 * time.Millisecond)

	proc.Stop(context.Background())

	assert.Len(t, called, 0)
}

func TestOpenAndClose(t *testing.T) {
	opened := make(chan struct{})
	closed := make(chan struct{})

	proc := NewProcess(logrus.New())

	proc.HandleFunc(&config.Process{
		Executable: "top",
		State:      config.Open,
	}, func() {
		opened <- struct{}{}
	})

	proc.HandleFunc(&config.Process{
		Executable: "top",
		State:      config.Close,
	}, func() {
		closed <- struct{}{}
	})

	go proc.Run()

	<-time.After(500 * time.Millisecond)

	proc.setRunning("top", true)

	select {
	case <-opened:
		t.Log("Open triggered")
	case <-time.After(1 * time.Second):
		panic("Timeout")
	}

	proc.setRunning("top", false)

	select {
	case <-closed:
		t.Log("Close triggered")
	case <-time.After(1 * time.Second):
		panic("Timeout")
	}

	proc.Stop(context.Background())
}

func TestStartStopDifferentGoRoutines(t *testing.T) {
	proc := NewProcess(logrus.New())
	go proc.Run()
	time.Sleep(2 * time.Millisecond)
	proc.Stop(context.Background())
}
