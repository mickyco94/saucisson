package config

type Condition string

const (
	FileKey    Condition = "file"
	CronKey    Condition = "cron"
	Processkey Condition = "process"
)

type Operation string

var (
	Create Operation = "create"
	Update Operation = "update"
	Remove Operation = "remove"
	Rename Operation = "remove"

	// Open  State = "open"
	// Close State = "close"
)

type Cron struct {
	Schedule string `yaml:"schedule"`
}

type File struct {
	Operation Operation `yaml:"operation"`
	Path      string    `yaml:"path"`
	Recursive bool      `yaml:"recursive"`
}

func (p *Process) Defaults() {}

type Process struct {
	Executable string `yaml:"executable"`
	State      string `yaml:"state"`
}
