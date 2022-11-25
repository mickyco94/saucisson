package main

import (
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/robfig/cron/v3"
	"gopkg.in/yaml.v3"
)

type BaseConfig struct {
	Services []ServiceConfig
}

//For every service config we want the execute to live
//in its own struct, that struct listens to a channel
//that the condition writes to
//condition can also live within its own struct
//has its own goroutine for that condition being satisfied

//We then spin up N of these for each definition within BaseConfig
//So the total number of goroutines is N * 2, where N is the no. of services

//Basic structure
// - parser (interprets the actual YAML)
// - service (composition of conditions + executors)
// - conditions
// - executors
// - cmd (need to figure out the daemon + cli aspect)
//

type ServiceConfig struct {
	Name      string
	Condition Condition
	Execute   Execute
}

//Move to raw, look at benthos
// type ConditionGeneric struct {
// 	v map[string]string
// }

type Condition struct {
	Cron CronCondition
}

type CronCondition struct {
	Schedule string
}

type Execute struct {
	Shell Shell
}

type Shell struct {
	Command string
}

func (s *Shell) getShell() string {
	return os.Getenv("SHELL")
}

func (s *Shell) execute() error {
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

func main() {
	cfg, err := parseConfig("./template.yml")

	if err != nil {
		log.Printf("err: %v\n", err)
		return
	}

	service := cfg.Services[0]

	conditionTrigger := make(chan struct{})
	c := cron.New(cron.WithSeconds())

	_, err = c.AddFunc(service.Condition.Cron.Schedule, func() {
		log.Printf("Triggering job")
		conditionTrigger <- struct{}{}
	})

	if err != nil {
		log.Printf("err: %v", err)
	}

	go func() {
		for {
			<-conditionTrigger
			service.Execute.Shell.execute()
		}
	}()

	if err != nil {
		log.Printf("err: %v\n", err)
	}

	c.Run()
}

// parseConfig attempts to read a config from the specified path
func parseConfig(path string) (*BaseConfig, error) {
	cfg := &BaseConfig{}

	bytes, err := os.ReadFile(path)

	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(bytes, cfg)

	if err != nil {
		return nil, err
	}

	return cfg, nil
}
