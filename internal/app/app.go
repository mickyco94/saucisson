package app

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/mickyco94/saucisson/internal/config"
	"github.com/mickyco94/saucisson/internal/executor"
	"github.com/mickyco94/saucisson/internal/service"
	"github.com/sirupsen/logrus"
)

type App struct {
	logger logrus.FieldLogger

	cron    *service.Cron
	file    *service.File
	process *service.Process
	pool    *executor.Pool
}

func New() *App {
	formatter := &logrus.TextFormatter{
		FullTimestamp: true,
	}

	logger := logrus.New()

	logger.SetFormatter(formatter)
	logger.SetLevel(logrus.DebugLevel)

	return &App{
		logger:  logger,
		pool:    executor.NewExecutorPool(logger),
		cron:    service.NewCron(),
		process: service.NewProcess(logger),
		file:    service.NewFile(logger),
	}
}

func (app *App) debuggr() {
	for {
		timer := time.NewTimer(5 * time.Second)

		app.logger.
			WithField("gr_count", runtime.NumGoroutine()).
			Debug("Debugging")

		<-timer.C
	}
}

func Run(templatePath string) error {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	app := New()

	file, err := os.Open(templatePath)
	if err != nil {
		return err
	}

	cfg := &config.Raw{}

	err = cfg.Parse(file)
	go app.debuggr()

	if err != nil {
		return err
	}

	for _, s := range cfg.Services {
		svc := app.construct(s)
		queueJob := func() {
			app.pool.Enqueue(executor.Job{
				Service:  s.Name,
				Executor: svc.executor,
			})
		}
		for _, fileCond := range svc.files {
			err := app.file.HandleFunc(fileCond, queueJob)

			if err != nil {
				panic(err)
			}
		}
		for _, cronCond := range svc.crons {
			app.cron.HandleFunc(cronCond, queueJob)
			if err != nil {
				panic(err)
			}
		}
		for _, processCond := range svc.processes {
			app.process.HandleFunc(processCond, queueJob)
			if err != nil {
				panic(err)
			}
		}
	}

	fileProccessorClosedChan := make(chan struct{})

	go func() {
		err := app.file.Run(time.Millisecond * 100)
		if err != nil {
			app.logger.
				WithError(err).
				Error("File proccessor shutdown unexpectedly")
			close(fileProccessorClosedChan)
		}
	}()

	go app.cron.Run()
	app.pool.Run()

	processRunnerClosedChan := make(chan struct{})
	go func() {
		err := app.process.Run()
		if err != nil {
			close(processRunnerClosedChan)
		}
	}()

	defer app.shutdown()

	select {
	case <-fileProccessorClosedChan:
		app.logger.Error("File service failed unexpectedly, shutting down")
	case <-sig:
		app.logger.Debug("Received SIGINT, shutting down")
	case <-processRunnerClosedChan:
		app.logger.Error("Process service failed unexpectedly, shutting down")
	}

	return nil
}

func (app *App) shutdown() {
	timer := time.AfterFunc(5*time.Second, func() {
		app.logger.Warn("Forcefully shutting down")
		os.Exit(0)
	})

	app.cron.Stop()
	app.file.Stop()
	app.pool.Stop()
	app.process.Stop()

	timer.Stop()
}

// definition can have any number of conditions of different types
// That all need to be registered
// For all of those conditions, each executor needs to be registered
type definition struct {
	crons     []*config.Cron
	files     []*config.File
	processes []*config.Process

	executor executor.Executor
}

// construct constructs an actual implementation of a Service from
// a specification
func (app *App) construct(spec config.ServiceSpec) *definition {
	svc := &definition{
		crons:    make([]*config.Cron, 0),
		files:    make([]*config.File, 0),
		executor: nil,
	}

	switch spec.Condition.Type {
	case config.CronKey:
		cronConf := &config.Cron{}
		spec.Condition.Config.Decode(cronConf)
		svc.crons = append(svc.crons, cronConf)
	case config.FileKey:
		fileConf := &config.File{}
		spec.Condition.Config.Decode(fileConf)
		svc.files = append(svc.files, fileConf)
	case config.Processkey:
		processConf := &config.Process{}
		spec.Condition.Config.Decode(processConf)
		svc.processes = append(svc.processes, processConf)
	}

	switch spec.Execute.Type {
	case "shell":
		//TODO: The context here should come from the executor pool
		//Probably not a func of the object. But of the `Execute()` method.
		shell := executor.NewShell(context.TODO(), app.logger)
		spec.Execute.Config.Decode(shell)
		svc.executor = shell
	}

	return svc
}
