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

func NewShell(
	logger logrus.FieldLogger) *Shell {
	return &Shell{
		logger:  logger,
		Timeout: 5,
	}
}

type Shell struct {
	logger logrus.FieldLogger

	Shell   string `yaml:"shell"`
	Command string `yaml:"command"`
	Timeout int    `yaml:"timeout"`
}

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

func escape(input string) string {
	return strings.Replace(input, "\"", "", -1)
}

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

	shell.logger.
		WithField("stdout", escape(string(out))).
		WithField("shell", sh).
		WithField("input", escape(shell.Command)).
		Info("Completed")

	return nil
}
