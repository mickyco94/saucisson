package component

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
)

func (c *ShellConfig) FromConfig(ctx context.Context, logger logrus.FieldLogger) (*Shell, error) {
	return NewShell(ctx, c.Command, logger), nil
}

func NewShell(
	ctx context.Context,
	command string,
	logger logrus.FieldLogger) *Shell {
	return &Shell{
		ctx:     ctx,
		logger:  logger,
		command: command,
	}
}

type Shell struct {
	ctx    context.Context
	logger logrus.FieldLogger

	command string
}

func (shell *Shell) getShell() string {
	val, exists := os.LookupEnv("SHELL")
	if !exists {
		return "bash" //? Make the aggressive assumption that they have bash
	}
	return val
}

func escape(input string) string {
	return strings.Replace(input, "\"", "", -1)
}

func (shell *Shell) Execute() error {

	sh := shell.getShell()

	//We should wait until the command is done, with a maximum timeout
	cmd, err := exec.CommandContext(shell.ctx, sh, "-c", escape(shell.command)).Output()

	if err != nil {
		return err
	}

	stdout := string(cmd)

	shell.logger.
		WithField("stdout", escape(stdout)).
		WithField("input", escape(shell.command)).
		Info("Completed")

	return nil

}

type ShellConfig struct {
	Command string `yaml:"command"`
}
