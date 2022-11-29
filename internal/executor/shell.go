package executor

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

func NewShell(
	ctx context.Context,
	logger logrus.FieldLogger) *Shell {
	return &Shell{
		ctx:    ctx,
		logger: logger,
	}
}

type ShellConfig struct {
	Command string `yaml:"command"`
}

func (sh *Shell) Configure(config yaml.Node) {
	cfg := &ShellConfig{}
	config.Decode(cfg)

	sh.Command = cfg.Command
}

type Shell struct {
	ctx    context.Context
	logger logrus.FieldLogger

	Command string
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
	cmd, err := exec.CommandContext(shell.ctx, sh, "-c", escape(shell.Command)).Output()

	if err != nil {
		return err
	}

	stdout := string(cmd)

	shell.logger.
		WithField("stdout", escape(stdout)).
		WithField("input", escape(shell.Command)).
		Info("Completed")

	return nil

}
