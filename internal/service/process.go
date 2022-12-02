package service

import (
	"time"

	"github.com/mickyco94/saucisson/internal/config"
	"github.com/mitchellh/go-ps"
)

type Process struct {
	entries  []processEntry
	watching map[string]struct{}
}

type processEntry struct {
	executable string
	listenFor  string //Listen for an open or close
	isRunning  bool
	h          func()
}

func NewProcess() *Process {
	return &Process{
		entries:  make([]processEntry, 0),
		watching: make(map[string]struct{}),
	}
}

func (p *Process) HandleFunc(config *config.Process, f func()) {
	entry := processEntry{
		executable: config.Executable,
		listenFor:  config.State,
		isRunning:  false,
		h:          f,
	}

	p.entries = append(p.entries, entry)
	p.watching[config.Executable] = struct{}{}
}

func (entry processEntry) startJob() {
	go entry.h()
}

var PollingInterval = 100 * time.Millisecond

func (p *Process) Run() error {
	for {
		if len(p.entries) == 0 {
			continue
		}

		timer := time.NewTimer(PollingInterval)

		processes, err := ps.Processes()

		if err != nil {
			//! Maybe have a retry on this..?
			//! p can keep track of the number of attempts
			//! If it succeeds then reset the counter
			//! Log.Debug on failure
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

			if isRunning && entry.listenFor == "open" && !entry.isRunning {
				entry.startJob()
			}

			if !isRunning && entry.listenFor == "close" && entry.isRunning {
				entry.startJob()
			}

			p.entries[i].isRunning = isRunning
		}

		<-timer.C
	}
}
