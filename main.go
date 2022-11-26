package main

import (
	"context"
	"errors"
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
	Services []*ServiceConfig
}

func (b *BaseConfig) Start() error {
	for _, v := range b.Services {
		err := v.startService()
		if err != nil {
			return err
		}
	}
	return nil
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

	name      string
	condition condition.Condition
	executor  executor.Executor
}

func NewServiceConfig(ctx context.Context,
	wg *sync.WaitGroup,
	name string,
	condition condition.Condition,
	executor executor.Executor) *ServiceConfig {
	return &ServiceConfig{
		ctx:       ctx,
		wg:        wg,
		name:      name,
		condition: condition,
		executor:  executor,
	}
}

//Move to raw, look at benthos
// type ConditionGeneric struct {
// 	v map[string]string
// }

// This is gross, should be &App{}
type App struct {
	cron    *cron.Cron
	context context.Context
}

func New(ctx context.Context) *App {
	return &App{
		context: ctx,
		cron:    cron.New(cron.WithSeconds()),
	}
}

func (d *App) conditionFactory(cond Raw) (condition.Condition, error) {
	componentName, err := cond.Name()

	if err != nil {
		return nil, err
	}

	if componentName == "cron" {
		sectionConfig, err := cond.ExtractSection("cron")
		if err != nil {
			return nil, err
		}
		sc, err := sectionConfig.ExtractString("schedule")
		if err != nil {
			return nil, err
		}

		impl := condition.NewCronCondition(sc, d.cron)

		return impl, nil
	}
	return nil, errors.New("Unsupported component")
}

func executorFactory(xc Raw) (executor.Executor, error) {
	componentName, err := xc.Name()

	if err != nil {
		return nil, err
	}

	if componentName == "shell" {
		sectionConfig, err := xc.ExtractSection("shell")
		if err != nil {
			return nil, err
		}
		comm, err := sectionConfig.ExtractString("command")
		if err != nil {
			return nil, err
		}

		impl := &executor.Shell{
			Command: comm,
		}

		return impl, nil
	}
	return nil, errors.New("Unsupported component")
}

func (a *App) Run() error {
	rawCfg, err := parseConfig("./template.yml")

	if err != nil {
		return err
	}
	// Basic application setup
	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		sig := make(chan os.Signal)
		signal.Notify(sig, os.Interrupt)
		<-sig
		fmt.Printf("Shutting down\n")
		cancel()
	}()

	//Service pipeline setup
	runtimeConfig := &BaseConfig{
		Services: make([]*ServiceConfig, len(rawCfg.Services)),
	}

	for i, v := range rawCfg.Services {
		xcImpl, err := executorFactory(v.Execute)
		condImpl, err := a.conditionFactory(v.Condition)

		if err != nil {
			return err
		}

		serviceConfig := &ServiceConfig{
			ctx:       ctx,
			wg:        wg,
			name:      v.Name,
			condition: condImpl,
			executor:  xcImpl,
		}
		runtimeConfig.Services[i] = serviceConfig
	}

	a.cron.Start()
	err = runtimeConfig.Start()

	if err != nil {
		return err
	}

	//Basic application setup tidy up
	wg.Wait()
	log.Printf("Stopping cron service")
	crnContext := a.cron.Stop()
	<-crnContext.Done()
	log.Printf("Cron stopped")

	return nil
}

func main() {
	New(context.Background()).Run()
}

func (s *ServiceConfig) startService() error {
	trigger := make(chan struct{})
	s.wg.Add(1)

	err := s.condition.Register(trigger)
	if err != nil {
		return err
	}

	//TODO: Error handling for execute
	go func(s *ServiceConfig) {
		defer func() {
			log.Printf("Service ending: %v\n", s.name)
			s.wg.Done()
		}()

		for {

			select {
			case <-trigger:
				err := s.executor.Execute()
				if err != nil {
					log.Fatalf("Error: %v\n", err)
				}
			case <-s.ctx.Done():
				return
			}

		}
	}(s)
	return nil
}

type RawConfig struct {
	Services []struct {
		Name      string
		Condition Raw
		Execute   Raw
	}
}

// Can be avoided by being more strongly typed
func (r Raw) Name() (string, error) {
	i := 0
	key := ""
	for k, _ := range r {
		if i > 0 {
			//TODO: Must be a better way than this lol
			return "", errors.New("Multiple keys specified for component declaration")
		}
		i++
		key = k
	}
	return key, nil
}

func (r Raw) ExtractString(key string) (string, error) {
	v, ok := r[key]
	if !ok {
		return "", errors.New("Key does not exist!!!")
	}
	return v.(string), nil
}

func (r Raw) ExtractSection(key string) (Raw, error) {
	v, ok := r[key]
	if !ok {
		return nil, errors.New("Section does not exist!")
	}
	return v.(Raw), nil
}

// ! This'll do for now
// Not the best interface to define but I can flesh this out
// When we come to defining a Linter
type Raw map[string]interface{}

type CronComponentConfiguration struct {
	Schedule string
}

// parseConfig attempts to read a config from the specified path
func parseConfig(path string) (*RawConfig, error) {
	rawConfig := &RawConfig{}

	bytes, err := os.ReadFile(path)

	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(bytes, rawConfig)

	if err != nil {
		return nil, err
	}

	return rawConfig, nil
}
