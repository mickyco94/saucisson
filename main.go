package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"time"

	"github.com/mickyco94/saucisson/condition"
	"github.com/mickyco94/saucisson/executor"
	"github.com/robfig/cron/v3"
	"gopkg.in/yaml.v3"
)

type BaseConfig struct {
	Services []*Service
}

//Basic structure
// - parser (interprets the actual YAML)
// - service (composition of conditions + executors)
// - conditions
// - executors
// - cmd (need to figure out the daemon + cli aspect)
//

// !This can be our domain level understandable thing..
type Service struct {
	Name      string
	Condition condition.Condition
	Executor  executor.Executor
}

func NewService(
	name string,
	condition condition.Condition,
	executor executor.Executor) *Service {

	return &Service{
		Name:      name,
		Condition: condition,
		Executor:  executor,
	}
}

type App struct {
	context      context.Context
	cron         *cron.Cron
	fileListener *condition.FileListener //Move FileListener into its own package, like cron. Or move them all into their own packages
}

func New(ctx context.Context) *App {
	return &App{
		context:      ctx,
		cron:         cron.New(cron.WithSeconds()),
		fileListener: condition.NewFileListener(ctx),
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
	} else if componentName == "file" {
		sectionConfig, err := cond.ExtractSection("file")
		if err != nil {
			return nil, err
		}
		path, err := sectionConfig.ExtractString("path")
		if err != nil {
			return nil, err
		}

		impl := condition.NewFile(path, d.fileListener)

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

type Job struct {
	serviceName string
	executor    executor.Executor
}

func (a *App) Run() error {
	rawCfg, err := parseConfig("./template.yml")

	if err != nil {
		return err
	}

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
			Name:      v.Name,
			Condition: condImpl,
			Executor:  xcImpl,
		}
		runtimeConfig.Services[i] = serviceConfig
	}

	jobs := make(chan *Job)

	for _, s := range runtimeConfig.Services {
		//Need to caputre s ref
		func(s *Service) {
			log.Printf("Registering: %v\n", s.Name)
			err := s.Condition.Register(func() {
				jobs <- &Job{
					serviceName: s.Name,
					executor:    s.Executor,
				}
			})

			if err != nil {
				log.Printf("Error registering: %v\n", err)
			}
		}(s)
	}

	worker := func(jobs chan *Job) {
		defer wg.Done()
		for j := range jobs {
			j.executor.Execute()
		}
	}

	go func() {
		for {
			log.Printf("GR: %v\n", runtime.NumGoroutine())
			time.Sleep(1 * time.Second)
		}
	}()

	workerCount := len(runtimeConfig.Services)
	//Max 20 GRs
	//Allow override with env variable
	if workerCount > 20 {
		workerCount = 20
	}

	wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go worker(jobs)
	}

	//Start all the services
	a.cron.Start()
	a.fileListener.Run()

	// if err != nil {
	// 	return err
	// }
	<-ctx.Done()

	//Stop producing...
	crnContext := a.cron.Stop()
	<-crnContext.Done()
	close(jobs)

	//Wait for all consumers to exit
	//Basic application setup tidy up
	wg.Wait()

	return nil
}

func main() {
	err := New(context.Background()).Run()
	if err != nil {
		log.Panicf("err: %v", err)
	}
}

func (s *Service) Logf(format string, v ...any) {
	args := []any{s.Name}
	if len(v) != 0 {
		args = append(args, v)
	}

	log.Printf("\033[31m %v\033[0m: "+format, args...)
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
	for k := range r {
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
