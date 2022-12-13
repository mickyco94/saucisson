package config

// Condition is the identifier for condition types that can be found in config:
type Condition string

const (
	FileKey    Condition = "file"
	CronKey    Condition = "cron"
	Processkey Condition = "process"
)

// Operation refers to the file operations that can be watched as part of the
// File condition
type Operation string

var (
	Create Operation = "create"
	Update Operation = "update"
	Remove Operation = "remove"
	Rename Operation = "rename"
)

// Cron defines the schedule for a cron based condition
type Cron struct {
	Schedule string `yaml:"schedule"`
}

// File defines the path and change operation applied to that path that
// the file condition should watch for
type File struct {
	Operation Operation `yaml:"operation"`
	Path      string    `yaml:"path"`
}

// State refers to the state change of a running process, i.e. open/close
type State string

var (
	Open  State = "open"
	Close State = "close"
)

// Process defines the configuration of the process change condition
// executable corresponds to the name of the process e.g. firefox.exe on Windows
type Process struct {
	Executable string `yaml:"executable"`
	State      State  `yaml:"state"`
}
