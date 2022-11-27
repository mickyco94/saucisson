package app

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"time"

	"github.com/mickyco94/saucisson/internal/condition"
	"github.com/mickyco94/saucisson/internal/executor"
	"github.com/mickyco94/saucisson/internal/parser"
	"github.com/robfig/cron/v3"
)

type ConditionFactoryFunc func(c parser.Raw, r Dependencies) (condition.Condition, error)
type ExecutorFactoryFunc func(c parser.Raw, r Dependencies) (executor.Executor, error)

// Registry holds all the factories
type Registry struct {
	conditions map[string]ConditionFactoryFunc
	executors  map[string]ExecutorFactoryFunc
}

func CronFactory(c parser.Raw, r Dependencies) (condition.Condition, error) {
	schedule, err := c.ExtractString("schedule")
	if err != nil {
		return nil, err
	}

	return condition.NewCronCondition(schedule, r.cron), nil
}

func FileListenerFactory(c parser.Raw, r Dependencies) (condition.Condition, error) {

	path, err := c.ExtractString("path")
	if err != nil {
		return nil, err
	}

	return condition.NewFile(path, r.fileListener), nil
}

func ShellExecutorFactory(c parser.Raw, r Dependencies) (executor.Executor, error) {
	command, err := c.ExtractString("command")

	if err != nil {
		return nil, err
	}

	return &executor.Shell{
		Command: command,
	}, nil
}

func NewRegistry() *Registry {
	return &Registry{
		conditions: map[string]ConditionFactoryFunc{
			"cron": CronFactory,
			"file": FileListenerFactory,
		},
		executors: map[string]ExecutorFactoryFunc{
			"shell": ShellExecutorFactory,
		},
	}
}

type BaseConfig struct {
	Services []*Service
}

type Dependencies struct {
	cron         *cron.Cron
	fileListener *condition.FileListener
}

type App struct {
	context      context.Context
	workerWg     *sync.WaitGroup
	Dependencies Dependencies
	Registry     *Registry
}

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

func (d *App) conditionFactory(cond parser.Raw) (condition.Condition, error) {
	componentName, err := cond.Name()

	if err != nil {
		return nil, err
	}

	constructor, exists := d.Registry.conditions[componentName]

	if !exists {
		return nil, errors.New("Component undefined")
	}

	configSection, err := cond.ExtractSection(componentName)

	return constructor(configSection, d.Dependencies)
}

func executorFactory(xc parser.Raw) (executor.Executor, error) {
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

func New(ctx context.Context) *App {
	return &App{
		context:  ctx,
		workerWg: &sync.WaitGroup{},
		Registry: NewRegistry(),
		Dependencies: Dependencies{
			cron:         cron.New(cron.WithSeconds()),
			fileListener: condition.NewFileListener(ctx),
		},
	}
}

func (a *App) debugGoroutines() {
	go func() {
		for {
			log.Printf("GR: %v\n", runtime.NumGoroutine())
			time.Sleep(1 * time.Second)
		}
	}()
}

func (a *App) spawnWorkers(workerCount int, jobs chan *Job) {

	worker := func(jobss chan *Job) {
		defer a.workerWg.Done()
		for j := range jobss {
			j.executor.Execute()
		}
	}

	//Max 20 GRs
	//Allow override with env variable
	if workerCount > 20 {
		workerCount = 20
	}

	a.workerWg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go worker(jobs)
	}

}

func (app *App) Run() error {
	rawCfg, err := parser.Parse("./template.yml")

	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		sig := make(chan os.Signal)
		signal.Notify(sig, os.Interrupt)
		<-sig
		cancel()
	}()

	app.debugGoroutines()

	//Service pipeline setup
	runtimeConfig := &BaseConfig{
		Services: make([]*Service, len(rawCfg.Services)),
	}

	for i, v := range rawCfg.Services {
		xcImpl, err := executorFactory(v.Execute)
		condImpl, err := app.conditionFactory(v.Condition)

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

	//Start all the consumers
	app.spawnWorkers(len(runtimeConfig.Services), jobs)

	//Start producers
	app.Dependencies.Start()

	//Listen for cancellation
	//Should be a select on multiple things really
	<-ctx.Done()

	app.Dependencies.Stop()
	close(jobs)

	//Wait for all consumers to exit
	app.workerWg.Wait()

	return nil
}

func (deps *Dependencies) Start() {
	deps.cron.Start()
	deps.fileListener.Run()
}

func (deps *Dependencies) Stop() {
	deps.fileListener.Stop()
	crnContext := deps.cron.Stop()
	<-crnContext.Done()
}
