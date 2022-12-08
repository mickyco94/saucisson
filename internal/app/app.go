package app

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/mickyco94/saucisson/internal/config"
	"github.com/mickyco94/saucisson/internal/executor"
	"github.com/mickyco94/saucisson/internal/watcher"
	"github.com/sirupsen/logrus"
)

type App struct {
	logger logrus.FieldLogger

	cron    *watcher.Cron
	file    *watcher.File
	process *watcher.Process
	pool    *executor.Pool
}

func new() *App {
	formatter := &logrus.JSONFormatter{
		PrettyPrint: true,
	}

	logger := logrus.New()

	logger.SetFormatter(formatter)
	logger.SetLevel(logrus.DebugLevel)

	return &App{
		logger:  logger,
		pool:    executor.NewExecutorPool(logger),
		cron:    watcher.NewCron(),
		process: watcher.NewProcess(logger),
		file:    watcher.NewFile(logger),
	}
}

func Run(templatePath string) error {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	app := new()

	file, err := os.Open(templatePath)
	if err != nil {
		return err
	}

	cfg := &config.Raw{}

	err = cfg.Parse(file)

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
		if svc.file != nil {
			err := app.file.HandleFunc(svc.file, queueJob)

			if err != nil {
				panic(err)
			}
		} else if svc.cron != nil {
			app.cron.HandleFunc(svc.cron, queueJob)
		} else if svc.process != nil {
			app.process.HandleFunc(svc.process, queueJob)
		} else {
			app.logger.WithField("svc", s.Name).Panic("Has no condition specified")
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

var shutdownDelay = time.Second * 5

func (app *App) shutdown() {
	wg := &sync.WaitGroup{}
	shutdownCtx, done := context.WithTimeout(context.Background(), shutdownDelay)
	defer done()

	wg.Add(1)
	go func() {
		defer wg.Done()

		err := app.file.Stop(shutdownCtx)
		if err != nil && err != watcher.ErrFileWatcherAlreadyClosed {
			app.logger.WithError(err).Error("File failed to shutdown")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		err := app.cron.Stop(shutdownCtx)
		if err != nil {
			app.logger.WithError(err).Error("Cron failed to shutdown")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		err := app.process.Stop(shutdownCtx)
		if err != nil {
			app.logger.WithError(err).Error("Process watcher failed to shutdown")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		err := app.pool.Stop(shutdownCtx)
		if err != nil {
			app.logger.WithError(err).Error("Executors failed to shutdown")
		}
	}()

	wg.Wait()
}

// definition can have any number of conditions of different types
// That all need to be registered
// For all of those conditions, each executor needs to be registered
type definition struct {
	cron    *config.Cron
	file    *config.File
	process *config.Process

	executor executor.Executor
}

// construct constructs an actual implementation of a Service from
// a specification
func (app *App) construct(spec config.ServiceSpec) *definition {
	svc := &definition{}

	switch spec.Condition.Type {
	case config.CronKey:
		cronConf := &config.Cron{}
		spec.Condition.Config.Decode(cronConf)
		svc.cron = cronConf
	case config.FileKey:
		fileConf := &config.File{}
		spec.Condition.Config.Decode(fileConf)
		svc.file = fileConf
	case config.Processkey:
		processConf := &config.Process{}
		spec.Condition.Config.Decode(processConf)
		svc.process = processConf
	}

	switch spec.Execute.Type {
	case "shell":
		shell := executor.NewShell(app.logger)
		spec.Execute.Config.Decode(shell)
		svc.executor = shell
	case "http":
		http := executor.NewHttp(app.logger)
		spec.Execute.Config.Decode(http)
		svc.executor = http
	}

	return svc
}
