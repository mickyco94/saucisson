package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/mickyco94/saucisson/condition"
	"github.com/mickyco94/saucisson/executor"
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
	ctx context.Context
	wg  *sync.WaitGroup

	Name      string
	Condition condition.Condition
	Executor  executor.Executor
}

//Move to raw, look at benthos
// type ConditionGeneric struct {
// 	v map[string]string
// }

func main() {
	wg := &sync.WaitGroup{}

	crn := cron.New(cron.WithSeconds())
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		sig := make(chan os.Signal)
		signal.Notify(sig, os.Interrupt)
		<-sig
		fmt.Printf("Shutting down\n")
		cancel()
	}()

	executor := &executor.Shell{
		Command: "echo hello world",
	}

	condition := condition.NewCronCondition("*/5 * * * * *", crn)

	svc := &ServiceConfig{
		wg:        wg,
		ctx:       ctx,
		Name:      "Billy",
		Condition: condition,
		Executor:  executor,
	}

	svc.startService()

	crn.Start()

	wg.Wait()
	log.Printf("Stopping cron service")
	crnContext := crn.Stop()
	<-crnContext.Done()
	log.Printf("Cron stopped")
}

func (s *ServiceConfig) startService() error {
	trigger := make(chan struct{})
	s.wg.Add(1)

	s.Condition.Register(trigger)

	//TODO: Error handling + cancellation etc.
	go func() {
		defer func() {
			log.Printf("Service ending: %v\n", s.Name)
			s.wg.Done()
		}()

		for {
			select {
			case <-trigger:
				{
					log.Printf("Executing service: %v\n", s.Name)
					s.Executor.Execute()
				}
			case <-s.ctx.Done():
				return
			}

		}
	}()

	return nil
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
