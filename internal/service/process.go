package service

import (
	"fmt"
	"time"

	"github.com/mickyco94/saucisson/internal/config"
	"github.com/mitchellh/go-ps"
)

// ? I could just give them all IDs :shrug:
type hashset map[string]processEntry

func (hashset hashset) add(pe processEntry) {
	hashset[pe.String()] = pe
}

func (hashset hashset) delete(pe processEntry) {
	delete(hashset, pe.String())
}

func (hashset hashset) Contains(pe processEntry) bool {
	_, exists := hashset[pe.String()]
	return exists
}

func (pe processEntry) String() string {
	//! Super wonky hash
	return fmt.Sprintf("%v_%v_%v", pe.h, pe.isRunning, pe.listenFor)
}

type Process struct {
	entries           map[string][]processEntry
	liteningForAClose hashset
	watching          hashset
}

type processEntry struct {
	executable string
	listenFor  string //Listen for an open or close
	isRunning  bool
	h          func()
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
		executable: config.Executable,
		listenFor:  config.State,
		isRunning:  false, //TODO: This will be set on `Run()`
		h:          f,
	}

	p.addEntry(config.Executable, entry)
	p.watching.add(entry)
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

func (entry processEntry) startJob() {
	go entry.h()
}

var PollingInterval = 100 * time.Millisecond

// Let the client decide to run on its own GR
func (p *Process) Run() error {

	//Initial state
	processes, _ := ps.Processes()
	for _, v := range processes {
		entries, exists := p.entries[v.Executable()]

		for _, entry := range entries {
			if exists {
				entry.isRunning = true

			}

		}
	}

	for {
		if len(p.entries) == 0 {
			//Skip if there is zero work to be done
			//Or return error..?
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

		//Split reading and dispatching into two GRs...?

		hs := make(hashset)

		for _, v := range p.entries {
			for _, e := range v {
				if e.listenFor == "close" && e.isRunning {
					hs.add(e)
				}
			}
		}

		//To allow having multiple keys we could just use an arr with
		//with executable as a prop on the struct and match against it
		//...
		//We could have a hashset of what we are watching to minimise throughput
		for _, process := range processes {
			entries := p.entries[process.Executable()]

			for i, entry := range entries {
				if !entry.isRunning && entry.listenFor == "open" {
					entry.startJob()
				} else if entry.isRunning && entry.listenFor == "close" {
					hs.delete(entry)
				}

				if !entry.isRunning {
					entry.isRunning = true
				}

				entries[i] = entry
				p.entries[process.Executable()] = entries
			}
		}

		//Any elements of the hs that remain are closed
		for _, k := range hs {
			k.startJob()
			entries := p.entries[k.executable]
			for i, entry := range entries {

				entry.isRunning = false
				entries[i] = entry
				p.entries[k.executable] = entries
			}
		}

		<-timer.C
	}
}
