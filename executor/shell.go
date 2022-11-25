package executor

import (
	"log"
	"os"
	"os/exec"
	"strings"
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
