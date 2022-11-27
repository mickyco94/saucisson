package component

import (
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/mickyco94/saucisson/internal/dependencies"
	"github.com/mickyco94/saucisson/internal/parser"
)

type Shell struct {
	Command string
}

func (s *Shell) getShell() string {
	val, exists := os.LookupEnv("SHELL")
	if !exists {
		return "bash" //Make the aggressive assumption that they have bash
	}
	return val
}

func (s *Shell) Execute() error {
	runCmd := s.Command

	removeQuotes := strings.Replace(runCmd, "\"", "", -1)

	sh := s.getShell()

	cmd, err := exec.Command(sh, "-c", removeQuotes).Output()

	if err != nil {
		return err
	}

	log.Println("Output:\n", string(cmd))

	return nil

}

func ShellExecutorFactory(c parser.Raw, r *dependencies.Dependencies) (Executor, error) {
	command, err := c.ExtractString("command")

	if err != nil {
		return nil, err
	}

	return &Shell{
		Command: command,
	}, nil
}
