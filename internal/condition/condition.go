package condition

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
)

type Cron struct {
	Schedule string `yaml:"schedule"`
}

type File struct {
	Operation Operation `yaml:"operation"`
	Path      string    `yaml:"path"`
	Recursive bool      `yaml:"recursive"`
}

type Process struct {
	Executable string
}

type State uint32

const (
	Open State = iota
	Close
)
