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

type BaseConfig struct {
	Services []*Service
}

type App struct {
	context      context.Context
	workerWg     *sync.WaitGroup
	cron         *cron.Cron
	fileListener *condition.FileListener
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

		cron:         cron.New(cron.WithSeconds()),
		fileListener: condition.NewFileListener(ctx),
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

func (a *App) Run() error {
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

	// a.debugGoroutines()

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

	//Start all the consumers
	a.spawnWorkers(len(runtimeConfig.Services), jobs)

	//Start producers
	a.cron.Start()
	a.fileListener.Run()

	//Listen for cancellation
	<-ctx.Done()

	//Wait for producers to stop
	crnContext := a.cron.Stop()
	<-crnContext.Done()
	close(jobs)

	//Wait for all consumers to exit
	a.workerWg.Wait()

	return nil
}
