package app

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"time"

	"github.com/mickyco94/saucisson/internal/component"
	"github.com/mickyco94/saucisson/internal/parser"
	"github.com/mickyco94/saucisson/internal/service"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

type BaseConfig struct {
	Services []*svc
}

type App struct {
	context      context.Context
	logger       logrus.FieldLogger
	workerWg     *sync.WaitGroup
	cron         *cron.Cron
	filelistener *service.FileListener
}

type svc struct {
	name      string
	condition component.Condition
	executor  component.Executor
	logger    logrus.FieldLogger
}

func (s svc) Start(jobs chan<- *job) {
	s.logger.Info("Registering")
	err := s.condition.Register(func() {
		jobs <- &job{
			serviceName: s.name,
			executor:    s.executor,
		}
	})

	if err != nil {
		s.logger.Error("Error registering: %v", err)
	}
}

func NewService(
	name string,
	condition component.Condition,
	executor component.Executor,
	logger logrus.FieldLogger) *svc {

	return &svc{
		name:      name,
		condition: condition,
		executor:  executor,
		logger:    logger,
	}
}

type job struct {
	serviceName string
	executor    component.Executor
}

func New(ctx context.Context) *App {
	formatter := &logrus.TextFormatter{
		FullTimestamp: true,
	}

	logger := logrus.New().WithField("app", "saucission")

	logger.Logger.SetFormatter(formatter)
	logger.Logger.SetLevel(logrus.DebugLevel)

	return &App{
		context:      ctx,
		logger:       logger,
		workerWg:     &sync.WaitGroup{},
		cron:         cron.New(cron.WithSeconds()),
		filelistener: service.NewFileListener(ctx, logger),
	}
}

func (a *App) debugGoroutines() {
	go func() {
		for {
			a.logger.WithField("count", runtime.NumGoroutine()).Debug("GoRoutine counter")
			time.Sleep(1 * time.Second)
		}
	}()
}

func (a *App) spawnWorkers(workerCount int, jobs chan *job) {

	worker := func(jobss chan *job, id int) {
		defer func() {
			a.logger.Debug("Stopping worker")
			a.workerWg.Done()
		}()

		for j := range jobss {

			err := j.executor.Execute()

			if err != nil {
				a.logger.
					WithField("svc", j.serviceName).
					WithField("err", err).
					Error("Error executing")
			}
		}
	}

	//Max 20 GRs
	//Allow override with env variable
	if workerCount > 20 {
		workerCount = 20
	}

	a.workerWg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go worker(jobs, i)
	}

}

func (a *App) ConditionFactory(spec parser.ComponentSpec) component.Condition {
	if spec.Type == "file" {
		file := component.NewFile(a.filelistener)

		file.Configure(spec.Config)
		return file
	}
	if spec.Type == "cron" {
		cron := component.NewCron(a.cron)
		cron.Configure(spec.Config)
		return cron
	}

	return nil
}

func (a *App) ExecutorFactory(spec parser.ComponentSpec) component.Executor {
	if spec.Type == "shell" {
		shell := component.NewShell(a.context, a.logger)
		shell.Configure(spec.Config)
		return shell
	}

	return nil
}

func (app *App) Run() error {
	file, err := os.Open("./template.yml")
	if err != nil {
		return err
	}

	config := &parser.RawConfig{}

	err = config.Parse(file)

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
		Services: make([]*svc, len(config.Services)),
	}

	for i, v := range config.Services {

		condition := app.ConditionFactory(v.Condition[0])
		executor := app.ExecutorFactory(v.Execute[0])

		serviceConfig := NewService(v.Name, condition, executor, app.logger)
		runtimeConfig.Services[i] = serviceConfig
	}

	jobs := make(chan *job)

	for _, s := range runtimeConfig.Services {
		s.Start(jobs)
	}

	//Start all the consumers
	app.spawnWorkers(len(runtimeConfig.Services), jobs)

	//Start producers
	app.filelistener.Run(time.Millisecond * 100)
	app.cron.Start()

	//Listen for cancellation
	//Should be a select on multiple things really
	<-ctx.Done()

	app.cron.Stop()
	app.filelistener.Stop()
	close(jobs)

	//Wait for all consumers to exit
	app.workerWg.Wait()

	return nil
}
