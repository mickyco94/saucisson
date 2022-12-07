package executor

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
)

func NewShell(
	ctx context.Context,
	logger logrus.FieldLogger) *Shell {
	return &Shell{
		ctx:    ctx,
		logger: logger,
	}
}

type Shell struct {
	ctx    context.Context
	logger logrus.FieldLogger

	Shell   string `yaml:"shell"`
	Command string `yaml:"command"`
}

func (shell *Shell) getShell() string {
	if shell.Shell != "" {
		return shell.Shell
	}

	val, exists := os.LookupEnv("SHELL")
	if !exists {
		return "bash" //? Make the aggressive assumption that they have bash
	}
	return val
}

func escape(input string) string {
	return strings.Replace(input, "\"", "", -1)
}

// TODO: Context here should come from what calls execute
func (shell *Shell) Execute() error {

	sh := shell.getShell()

	//We should wait until the command is done, with a maximum timeout
	cmd, err := exec.CommandContext(shell.ctx, sh, "-c", escape(shell.Command)).Output()

	if err != nil {
		return err
	}

	stdout := string(cmd)

	shell.logger.
		WithField("stdout", escape(stdout)).
		WithField("shell", sh).
		WithField("input", escape(shell.Command)).
		Info("Completed")

	return nil

}
