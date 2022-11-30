package service

import (
	"time"

	"github.com/mickyco94/saucisson/internal/config"
	"github.com/mitchellh/go-ps"
)

type hashset map[string]struct{}

func (h hashset) add(key string) {
	h[key] = struct{}{}
}

type Process struct {
	//This is a dict of processes to watch and executors to run
	entries           map[string]processEntry
	liteningForAClose hashset
}

type processEntry struct {
	listenFor string //Listen for an open or close
	isRunning bool
	h         func()
}

func NewProcess() *Process {
	return &Process{
		entries:           make(map[string]processEntry),
		liteningForAClose: make(hashset),
	}
}

// TODO:
// Executable name is not perfect as a key, since multiple services could be listening
// to the same process with different executors
// e.g. onclose do x, onopen do y
// The service defines the uniqueness, not some parameter
func (p *Process) HandleFunc(config *config.Process, f func()) {
	//Some validaiton may be nice :)
	p.entries[config.Executable] = processEntry{
		listenFor: config.State,
		isRunning: false, //TODO: This will be set on `Run()`
		h:         f,
	}
}

func (entry *processEntry) SetRunning(isRunning bool) {
	entry.isRunning = isRunning
}

func (p *Process) init() {
	processes, _ := ps.Processes()
	for _, v := range processes {
		entry, exists := p.entries[v.Executable()]
		if exists {
			entry.isRunning = true

			if entry.listenFor == "close" {
				p.liteningForAClose.add(v.Executable())
			}
		}
	}
}

// Let the client decide to run on its own GR
func (p *Process) Run() error {

	//TODO: Get the initial state of the system at the time of running
	//TODO: That acts as the base for open/close operations. Otherwise we run
	//TODO: the executor as long as the process has been open. Need a more complex
	//TODO: struct and probably an init like method that can be run if HandleFunc
	//TODO: is called on a running process server

	p.init()

	for {
		delayChan := time.NewTimer(100 * time.Millisecond).C

		processes, err := ps.Processes()

		if err != nil {
			//! Maybe have a retry on this..?
			//! p can keep track of the number of attempts
			//! If it succeeds then reset the counter
			//! Log.Debug on failure
			return err
		}

		//Split reading and dispatching into two GRs...?
		//To listen for the close event we need to account for misses
		//Basically if entry hit count != entry count then something closed
		//Not sure the most efficient way to account for this.

		//Can probably have two different data structures for close vs. open events
		//Use a hashset for holding processes that are open currently and listening for a close
		//Create the hashset here, remove based on process.Executable(). Iterate over remaining

		hs := make(hashset)

		for k, v := range p.entries {
			if v.listenFor == "close" && v.isRunning {
				hs.add(k)
			}
		}

		for _, process := range processes {
			entry, watching := p.entries[process.Executable()]

			//? Separate the updating of state and checking of state :)

			//If the process is running, we don't know it's running and the entry
			//says to listen for a run. Then trigger
			if watching && !entry.isRunning && entry.listenFor == "open" {
				entry.h()
				entry.SetRunning(true)
				p.entries[process.Executable()] = entry
			} else if watching && entry.isRunning && entry.listenFor == "close" {
				delete(hs, process.Executable())
			}

			if !entry.isRunning {
				entry.SetRunning(true)
			}
			p.entries[process.Executable()] = entry
		}

		//Any elements of the hs that remain are closed
		for k := range hs {
			entry := p.entries[k]
			entry.h()
			entry.SetRunning(false)
			p.entries[k] = entry
		}

		//Wait for the delay
		<-delayChan
	}
}
