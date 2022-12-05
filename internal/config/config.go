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
	Rename Operation = "rename"
)

type Cron struct {
	Schedule string `yaml:"schedule"`
}

type File struct {
	Operation Operation `yaml:"operation"`
	Path      string    `yaml:"path"`
	Recursive bool      `yaml:"recursive"`
}

type State string

var (
	Open  State = "open"
	Close State = "close"
)

type Process struct {
	Executable string `yaml:"executable"`
	State      State  `yaml:"state"`
}
