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
	//The arr of entries will most likely always be of size one
	entries           map[string][]processEntry
	liteningForAClose hashset
	watching          hashset
}

type processEntry struct {
	listenFor string //Listen for an open or close
	isRunning bool
	h         func()
}

func NewProcess() *Process {
	return &Process{
		entries:           make(map[string][]processEntry),
		liteningForAClose: make(hashset),
		watching:          make(hashset),
	}
}

// TODO:
// Executable name is not perfect as a key, since multiple services could be listening
// to the same process with different executors
// e.g. onclose do x, onopen do y
// The service defines the uniqueness, not some parameter
func (p *Process) HandleFunc(config *config.Process, f func()) {
	//Some validaiton may be nice :)
	entry := processEntry{
		listenFor: config.State,
		isRunning: false, //TODO: This will be set on `Run()`
		h:         f,
	}

	p.addEntry(config.Executable, entry)
	p.watching.add(config.Executable)
}

func (p *Process) addEntry(key string, entry processEntry) {
	arr, exists := p.entries[key]
	if !exists {
		coll := []processEntry{entry}
		p.entries[key] = coll
	} else {
		arr = append(arr, entry)
		p.entries[key] = arr
	}
}

func (p *Process) init() {
	processes, _ := ps.Processes()
	for _, v := range processes {
		entries, exists := p.entries[v.Executable()]

		for _, entry := range entries {
			if exists {
				entry.isRunning = true

				if entry.listenFor == "close" {
					p.liteningForAClose.add(v.Executable())
				}
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
		if len(p.entries) == 0 {
			//Skip if there is zero work to be done
			continue
		}

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

		hs := make(hashset)

		for k, v := range p.entries {
			for _, e := range v {
				if e.listenFor == "close" && e.isRunning {
					hs.add(k)
				}
			}
		}

		//To allow having multiple keys we could just use an arr with
		//with executable as a prop on the struct and match against it
		//...
		//We could have a hashset of what we are watching to minimise throughput
		for _, process := range processes {
			entries, watching := p.entries[process.Executable()]

			for i, entry := range entries {
				if watching && !entry.isRunning && entry.listenFor == "open" {
					entry.h()
					//Update entry in place
					entry.isRunning = true
					entries[i] = entry
					p.entries[process.Executable()] = entries
				} else if watching && entry.isRunning && entry.listenFor == "close" {
					delete(hs, process.Executable())
				}

				if !entry.isRunning {
					entry.isRunning = true
				}
				p.entries[process.Executable()] = entries
			}

		}

		//Any elements of the hs that remain are closed
		for k := range hs {
			entries := p.entries[k]
			for i, entry := range entries {
				entry.h()
				entry.isRunning = true

				//Update the entry in place
				entries[i] = entry
				p.entries[k] = entries
			}
		}

		//Wait for the delay
		<-delayChan
	}
}
