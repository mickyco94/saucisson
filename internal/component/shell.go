package component

import (
	"os"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
)

func (c *ShellConfig) FromConfig(logger logrus.FieldLogger) (*Shell, error) {
	return NewShell(c.Command, logger), nil
}

func NewShell(
	command string,
	logger logrus.FieldLogger) *Shell {
	return &Shell{
		logger:  logger,
		command: command,
	}
}

type Shell struct {
	logger  logrus.FieldLogger
	command string
}

func (s *Shell) getShell() string {
	val, exists := os.LookupEnv("SHELL")
	if !exists {
		return "bash" //? Make the aggressive assumption that they have bash
	}
	return val
}

func escape(input string) string {
	return strings.Replace(input, "\"", "", -1)
}

func (s *Shell) Execute() error {

	sh := s.getShell()

	cmd, err := exec.Command(sh, "-c", escape(s.command)).Output()

	if err != nil {
		return err
	}

	stdout := string(cmd)

	s.logger.
		WithField("stdout", escape(stdout)).
		WithField("input", escape(s.command)).
		Info("Completed")

	return nil

}

type ShellConfig struct {
	Command string `yaml:"command"`
}
