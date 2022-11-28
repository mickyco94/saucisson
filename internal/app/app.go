package app

import (
	"context"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"time"

	"github.com/mickyco94/saucisson/internal/component"
	"github.com/mickyco94/saucisson/internal/parser"
	"github.com/mickyco94/saucisson/internal/service"
	"github.com/robfig/cron/v3"
)

type BaseConfig struct {
	Services []*Service
}

type App struct {
	context      context.Context
	workerWg     *sync.WaitGroup
	cron         *cron.Cron
	filelistener *service.FileListener
}

type Service struct {
	Name      string
	Condition component.Condition
	Executor  component.Executor
}

func NewService(
	name string,
	condition component.Condition,
	executor component.Executor) *Service {

	return &Service{
		Name:      name,
		Condition: condition,
		Executor:  executor,
	}
}

type Job struct {
	serviceName string
	executor    component.Executor
}

func New(ctx context.Context) *App {
	return &App{
		context:      ctx,
		workerWg:     &sync.WaitGroup{},
		cron:         cron.New(cron.WithSeconds()),
		filelistener: service.NewFileListener(ctx),
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

	// if 1 > 0 {
	// 	bytes, _ := json.MarshalIndent(rawCfg, "", "  ")
	// 	log.Printf("CFG: %v\n", string(bytes))
	// 	return nil
	// }

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

	// app.debugGoroutines()

	//Service pipeline setup
	runtimeConfig := &BaseConfig{
		Services: make([]*Service, len(rawCfg.Services)),
	}

	for i, v := range rawCfg.Services {
		var condition component.Condition
		var executor component.Executor

		//TODO: Support multiple conditions and executors
		if v.Condition.Cron != nil {
			condition, err = v.Condition.Cron.FromConfig(app.cron)
			if err != nil {
				return err
			}
		} else if v.Condition.File != nil {
			condition, err = v.Condition.File.FromConfig(app.filelistener)
			if err != nil {
				return err
			}
		}

		if v.Execute.Shell != nil {
			executor, err = v.Execute.Shell.FromConfig()
			if err != nil {
				return err
			}
		}

		serviceConfig := &Service{
			Name:      v.Name,
			Condition: condition,
			Executor:  executor,
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
	app.filelistener.Run()
	app.cron.Start()

	//Listen for cancellation
	//Should be a select on multiple things really
	<-ctx.Done()

	close(jobs)

	//Wait for all consumers to exit
	app.workerWg.Wait()

	return nil
}
