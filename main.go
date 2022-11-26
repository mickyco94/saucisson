package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/mickyco94/saucisson/condition"
	"github.com/mickyco94/saucisson/executor"
	"github.com/robfig/cron/v3"
	"gopkg.in/yaml.v3"
)

// ! RE-DESIGN TIME
// I suck at multi-threading, a short story.
// So what I currently have is pretty inefficient, we're holding executors and
// conditions in memory constantly and spawning threads for each services.
// Thinking about this, a better design for multi-threading and memory constraints
// We would be to have a limited number of goroutines that check for run conditions to
// be satisfied. Then from a pool of available executor threads we spawn and attach an executor
// This limits the number of threads that are running at any given time. We can try to make this
// equal to the number of services that have been defined but attach an upper bound

// The real question to me is how do we have a limited number of producer/condition check threads

// Side note: Channel go-between for communication between condition and executor needs to go.
// Service -> Executor
// Service -> Condition
// Hierarchy too, executors and conditions should have knowledge of their defining services so they know
// what executor to create/reference

type BaseConfig struct {
	Services []*Service
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

//Basic structure
// - parser (interprets the actual YAML)
// - service (composition of conditions + executors)
// - conditions
// - executors
// - cmd (need to figure out the daemon + cli aspect)
//

type Service struct {
	Name string

	ctx       context.Context
	wg        *sync.WaitGroup
	condition condition.Condition
	executor  executor.Executor
}

func NewService(
	name string,
	ctx context.Context,
	wg *sync.WaitGroup,
	condition condition.Condition,
	executor executor.Executor) *Service {

	return &Service{
		ctx:       ctx,
		wg:        wg,
		Name:      name,
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
		cancel()
	}()

	//Service pipeline setup
	runtimeConfig := &BaseConfig{
		Services: make([]*Service, len(rawCfg.Services)),
	}

	for i, v := range rawCfg.Services {
		xcImpl, err := executorFactory(v.Execute)
		condImpl, err := a.conditionFactory(v.Condition)

		if err != nil {
			return err
		}

		serviceConfig := &Service{
			ctx:       ctx,
			wg:        wg,
			Name:      v.Name,
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
	crnContext := a.cron.Stop()
	<-crnContext.Done()
	return nil
}

func main() {
	New(context.Background()).Run()
}

func (s *Service) Logf(format string, v ...any) {
	args := []any{s.Name}
	if len(v) != 0 {
		args = append(args, v)
	}

	log.Printf("\033[31m %v\033[0m: "+format, args...)
}

func (s *Service) startService() error {
	trigger := make(chan struct{})
	s.wg.Add(1)

	err := s.condition.Register(trigger)
	if err != nil {
		return err
	}

	//Right now, the number of goroutines scales with the YAML
	//file that we are using
	//We should apply some level of limiting...
	//Producer/Consumer model

	// The number of goroutines should be fixed and the scheduler should
	// take advantage of what is available...
	go func(s *Service) {
		defer func() {
			s.Logf("Exited")
			s.wg.Done()
		}()

		for {

			select {
			case <-trigger:
				err := s.executor.Execute()
				if err != nil {
					s.Logf("rut ro")
				}
			case <-s.ctx.Done():
				s.Logf("Exiting...")
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
