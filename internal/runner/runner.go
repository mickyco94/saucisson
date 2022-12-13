package runner

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/mickyco94/saucisson/internal/config"
	"github.com/mickyco94/saucisson/internal/executor"
	"github.com/mickyco94/saucisson/internal/watcher"
	"github.com/sirupsen/logrus"
)

// Runner is a service layer struct that corresponds
// to the saucisson run command.
// This service is responsible for instantiating dependencies,
// interpreting the provided configuration and coordinating those
// dependencies.
type Runner struct {
	logger logrus.FieldLogger

	cron    *watcher.Cron
	file    *watcher.File
	process *watcher.Process
	pool    *executor.Pool
}

// Run constructs and invokes a runner using the provided templatePath
// to retrieve the config that drives runner.
// Run will block and execute until a SIGINT signal is received from the os
// at which point Run will attempt to gracefully shutdown its dependencies.
func Run(templatePath string) error {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	formatter := &logrus.JSONFormatter{
		PrettyPrint: true,
	}

	logger := logrus.New()

	logger.SetFormatter(formatter)
	logger.SetLevel(logrus.DebugLevel)

	runner := &Runner{
		logger:  logger,
		pool:    executor.NewPool(logger, executor.DefaultPoolSize),
		cron:    watcher.NewCron(),
		process: watcher.NewProcess(logger),
		file:    watcher.NewFile(logger),
	}

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
		def := runner.construct(s)
		serviceName := s.Name
		queueJob := func() {
			runner.pool.Enqueue(executor.Job{
				Service:  serviceName,
				Executor: def.executor.Execute,
			})
		}
		if def.file != nil {
			err := runner.file.HandleFunc(def.file, queueJob)

			if err != nil {
				panic(err)
			}
		} else if def.cron != nil {
			err := runner.cron.HandleFunc(def.cron, queueJob)
			if err != nil {
				panic(err)
			}
		} else if def.process != nil {
			runner.process.HandleFunc(def.process, queueJob)
		} else {
			runner.logger.
				WithField("svc", s.Name).
				Panic("Has no condition specified")

			return nil
		}
	}

	fileProccessorClosedChan := make(chan struct{})

	go func() {
		err := runner.file.Run(time.Millisecond * 100)
		if err != nil {
			close(fileProccessorClosedChan)
		}
	}()

	go runner.cron.Run()
	runner.pool.Start()

	processRunnerClosedChan := make(chan struct{})
	go func() {
		err := runner.process.Run()
		if err != nil {
			close(processRunnerClosedChan)
		}
	}()

	defer runner.shutdown()

	select {
	case <-fileProccessorClosedChan:
		runner.logger.Error("File service failed unexpectedly, shutting down")
	case <-sig:
		runner.logger.Debug("Received SIGINT, shutting down")
	case <-processRunnerClosedChan:
		runner.logger.Error("Process service failed unexpectedly, shutting down")
	}

	return nil
}

// shutdownDelay is the maximum time limit dependent services have to exit.
// If this time limit is exceeded the application exits without properly
// terminating those dependencies.
var shutdownDelay = time.Second * 5

// shutdown closes the Runner by attempting to gracefully close all dependent services
func (runner *Runner) shutdown() {
	wg := &sync.WaitGroup{}
	shutdownCtx, done := context.WithTimeout(context.Background(), shutdownDelay)
	defer done()

	wg.Add(1)
	go func() {
		defer wg.Done()

		err := runner.file.Stop(shutdownCtx)
		if err != nil {
			runner.logger.WithError(err).Error("File watcher failed to shutdown")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		err := runner.cron.Stop(shutdownCtx)
		if err != nil {
			runner.logger.WithError(err).Error("Cron failed to shutdown")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		err := runner.process.Stop(shutdownCtx)
		if err != nil {
			runner.logger.WithError(err).Error("Process watcher failed to shutdown")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		err := runner.pool.Stop(shutdownCtx)
		if err != nil {
			runner.logger.WithError(err).Error("Executors failed to shutdown")
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
func (runner *Runner) construct(spec config.ServiceSpec) *definition {
	def := &definition{}

	switch spec.Condition.Type {
	case config.CronKey:
		cronConf := &config.Cron{}
		spec.Condition.Config.Decode(cronConf)
		def.cron = cronConf
	case config.FileKey:
		fileConf := &config.File{}
		spec.Condition.Config.Decode(fileConf)
		def.file = fileConf
	case config.Processkey:
		processConf := &config.Process{}
		spec.Condition.Config.Decode(processConf)
		def.process = processConf
	}

	switch spec.Execute.Type {
	case "shell":
		shell := executor.NewShell(runner.logger)
		spec.Execute.Config.Decode(shell)
		def.executor = shell
	case "http":
		http := executor.NewHttp(runner.logger, *http.DefaultClient) //TODO: This should be more specific..
		spec.Execute.Config.Decode(http)
		def.executor = http
	}

	return def
}
