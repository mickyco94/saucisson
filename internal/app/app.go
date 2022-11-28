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

func NewService(
	name string,
	condition component.Condition,
	executor component.Executor) *svc {

	return &svc{
		name:      name,
		condition: condition,
		executor:  executor,
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

	logger := logrus.New().
		WithField("app", "saucission").
		WithField("gr_count", runtime.NumGoroutine())

	logger.Logger.SetFormatter(formatter)

	logger.Logger.SetLevel(logrus.DebugLevel)
	return &App{
		context:      ctx,
		logger:       logger,
		workerWg:     &sync.WaitGroup{},
		cron:         cron.New(cron.WithSeconds()),
		filelistener: service.NewFileListener(ctx),
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

	worker := func(jobss chan *job) {
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

	// app.debugGoroutines()

	//Service pipeline setup
	runtimeConfig := &BaseConfig{
		Services: make([]*svc, len(rawCfg.Services)),
	}

	for i, v := range rawCfg.Services {
		var condition component.Condition
		var executor component.Executor

		//TODO: Support multiple conditions and executors
		//This pattern doesn't really scale..
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
			executor, err = v.Execute.Shell.FromConfig(app.logger)
			if err != nil {
				return err
			}
		}

		serviceConfig := &svc{
			name:      v.Name,
			condition: condition,
			executor:  executor,
			logger:    app.logger.WithField("svc", v.Name),
		}
		runtimeConfig.Services[i] = serviceConfig
	}

	jobs := make(chan *job)

	for _, s := range runtimeConfig.Services {
		//Need to caputre s ref
		func(s *svc) {
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
