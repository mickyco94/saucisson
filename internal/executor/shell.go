package executor

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// NewShell creates a new Shell based executor with default
// values set for optional fields in the configuration and all
// dependencies
func NewShell(logger logrus.FieldLogger) *Shell {
	return &Shell{
		logger:    logger,
		Timeout:   5,
		LogOutput: false,
	}
}

// Shell defines an execution that will run in the
// users defined shell. The command that will be run is specified
// via the Command member of this struct.
//
// Defaults:
// - Logging output is disabled
// - Timeout for commands is 5s
type Shell struct {
	logger logrus.FieldLogger

	LogOutput bool   `yaml:"log"`
	Shell     string `yaml:"shell"`
	Command   string `yaml:"command"`
	Timeout   int    `yaml:"timeout"`
}

// getShell determines the shell to use for execution of the specified
// command. This is determined either by user configuration or environment variables.
func (shell *Shell) getShell() string {
	if shell.Shell != "" {
		return shell.Shell
	}

	s, exists := os.LookupEnv("SHELL")
	if !exists {
		return "bash" //? Make the aggressive assumption that they have bash
	}
	return s
}

// escape simply removes all \ characters from the string
func escape(input string) string {
	return strings.Replace(input, "\"", "", -1)
}

// Execute runs command defined by Shell, using configuration that is provided
// by members of the defining struct
// ctx is used to propagate any cancellation instructions of the command from the caller
func (shell *Shell) Execute(ctx context.Context) error {
	ctx, done := context.WithTimeout(ctx, time.Second*time.Duration(shell.Timeout))
	defer done()

	sh := shell.getShell()

	out, err := exec.CommandContext(ctx, sh, "-c", escape(shell.Command)).Output()

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
			return ErrTimeoutExceeded
		}
		return err
	}

	if shell.LogOutput {
		shell.logger.
			WithField("stdout", escape(string(out))).
			WithField("shell", sh).
			WithField("input", escape(shell.Command)).
			Info("Shell execution output")
	}

	return nil
}
