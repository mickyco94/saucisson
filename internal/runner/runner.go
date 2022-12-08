package runner

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

type Runner struct {
	logger logrus.FieldLogger

	cron    *watcher.Cron
	file    *watcher.File
	process *watcher.Process
	pool    *executor.Pool
}

func new() *Runner {
	formatter := &logrus.JSONFormatter{
		PrettyPrint: true,
	}

	logger := logrus.New()

	logger.SetFormatter(formatter)
	logger.SetLevel(logrus.DebugLevel)

	return &Runner{
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

	runner := new()

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
		svc := runner.construct(s)
		queueJob := func() {
			runner.pool.Enqueue(executor.Job{
				Service:  s.Name,
				Executor: svc.executor,
			})
		}
		if svc.file != nil {
			err := runner.file.HandleFunc(svc.file, queueJob)

			if err != nil {
				panic(err)
			}
		} else if svc.cron != nil {
			err := runner.cron.HandleFunc(svc.cron, queueJob)
			if err != nil {
				panic(err)
			}
		} else if svc.process != nil {
			runner.process.HandleFunc(svc.process, queueJob)
		} else {
			runner.logger.WithField("svc", s.Name).Panic("Has no condition specified")
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
	runner.pool.Run()

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

var shutdownDelay = time.Second * 5

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
		http := executor.NewHttp(runner.logger)
		spec.Execute.Config.Decode(http)
		def.executor = http
	}

	return def
}
