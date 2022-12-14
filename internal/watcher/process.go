package watcher

import (
	"context"
	"sync"
	"time"

	"github.com/mickyco94/saucisson/internal/config"
	"github.com/mitchellh/go-ps"
	"github.com/sirupsen/logrus"
)

type State uint32

// Op
const (
	Open State = iota
	Close
)

type Processes func() ([]ps.Process, error)

type Process struct {
	source Processes

	logger logrus.FieldLogger

	runningMu sync.Mutex
	done      chan struct{}
	close     chan struct{}
	running   bool

	entries  []processEntry
	watching map[string]struct{}
}

type processEntry struct {
	executable string
	listenFor  State
	isRunning  bool
	h          func()
}

func NewProcess(logger logrus.FieldLogger) *Process {
	return &Process{
		source:    ps.Processes, //Setting this here supports mocking
		logger:    logger,
		runningMu: sync.Mutex{},
		done:      make(chan struct{}),
		close:     make(chan struct{}),
		running:   false,
		entries:   make([]processEntry, 0),
		watching:  make(map[string]struct{}),
	}
}

var stateStringToEnum = map[config.State]State{
	config.Close: Close,
	config.Open:  Open,
}

func (p *Process) HandleFunc(config *config.Process, f func()) {

	entry := processEntry{
		executable: config.Executable,
		listenFor:  stateStringToEnum[config.State],
		isRunning:  false,
		h:          f,
	}

	p.entries = append(p.entries, entry)
	p.watching[config.Executable] = struct{}{}
}

func (entry processEntry) startJob() {
	go entry.h()
}

func (p *Process) processes() ([]ps.Process, error) {
	backoff := 1

	for {
		procs, err := p.source()
		if err == nil {
			return procs, nil
		}

		if backoff > 32 {
			return nil, err
		}

		p.logger.
			WithError(err).
			Debug("Retrying fetching proccesses")

		time.Sleep(time.Duration(backoff * int(time.Second)))

		backoff *= 2
	}
}

var pollingInterval = 100 * time.Millisecond

func (p *Process) setInitialState() error {
	if len(p.entries) == 0 {
		//No state to set
		return nil
	}

	processes, err := p.processes()

	if err != nil {
		return err
	}

	runningProcs := make(map[string]struct{})

	for _, process := range processes {
		_, watching := p.watching[process.Executable()]
		//perf: Micro-optimisation to make runningProcs small as possible
		if watching {
			runningProcs[process.Executable()] = struct{}{}
		}
	}

	for i, entry := range p.entries {
		_, isRunning := runningProcs[entry.executable]

		p.entries[i].isRunning = isRunning
	}

	return nil
}

func (p *Process) Run() error {
	p.runningMu.Lock()
	if p.running {
		p.runningMu.Unlock()
		return nil
	}

	p.running = true
	p.runningMu.Unlock()

	err := p.setInitialState()

	if err != nil {
		return err
	}

	return p.run()
}

func (p *Process) run() error {
	for {
		select {
		case <-p.close:
			p.done <- struct{}{}
			return nil
		case <-time.After(pollingInterval):
			if len(p.entries) == 0 {
				continue
			}

			processes, err := p.processes()

			if err != nil {
				return err
			}

			runningProcs := make(map[string]struct{})

			for _, process := range processes {
				_, watching := p.watching[process.Executable()]
				//perf: Micro-optimisation to make runningProcs small as possible
				if watching {
					runningProcs[process.Executable()] = struct{}{}
				}
			}

			for i, entry := range p.entries {
				_, isRunning := runningProcs[entry.executable]

				if isRunning && entry.listenFor == Open && !entry.isRunning {
					entry.startJob()
				}

				if !isRunning && entry.listenFor == Close && entry.isRunning {
					entry.startJob()
				}

				p.entries[i].isRunning = isRunning
			}
		}
	}
}

// Stop signals to the main goroutine to halt processing and exit
// this method also waits for the main goroutine to signal that
// it has successfully closed.
// If `Process` is already stopped then this noops
func (proc *Process) Stop(ctx context.Context) error {

	if !proc.running {
		return nil
	}

	proc.close <- struct{}{}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-proc.done:
		return nil
	}
}
