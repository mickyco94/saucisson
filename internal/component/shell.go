package component

import (
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/mickyco94/saucisson/internal/parser"
)

func NewShell(command string) *Shell {
	return &Shell{
		command: command,
	}
}

type Shell struct {
	command string
}

func (s *Shell) getShell() string {
	val, exists := os.LookupEnv("SHELL")
	if !exists {
		return "bash" //Make the aggressive assumption that they have bash
	}
	return val
}

func (s *Shell) Execute() error {
	runCmd := s.command

	removeQuotes := strings.Replace(runCmd, "\"", "", -1)

	sh := s.getShell()

	cmd, err := exec.Command(sh, "-c", removeQuotes).Output()

	if err != nil {
		return err
	}

	log.Println("Output:\n", string(cmd))

	return nil

}

func ParseConfig(c parser.Raw) (Executor, error) {
	command, err := c.ExtractString("command")

	if err != nil {
		return nil, err
	}

	return &Shell{
		command: command,
	}, nil
}
