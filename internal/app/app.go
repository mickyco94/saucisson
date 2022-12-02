package app

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/mickyco94/saucisson/internal/config"
	"github.com/mickyco94/saucisson/internal/executor"
	"github.com/mickyco94/saucisson/internal/service"
	"github.com/sirupsen/logrus"
)

type App struct {
	context context.Context
	logger  logrus.FieldLogger

	cron    *service.Cron
	file    *service.File
	process *service.Process

	pool *executor.Pool
}

func New(ctx context.Context) *App {
	formatter := &logrus.TextFormatter{
		FullTimestamp: true,
	}

	logger := logrus.New()

	logger.SetFormatter(formatter)
	logger.SetLevel(logrus.DebugLevel)

	return &App{
		context: ctx,
		pool:    executor.NewExecutorPool(logger, 10),
		logger:  logger,
		cron:    service.NewCron(),
		process: service.NewProcess(),
		file:    service.NewFile(ctx, logger),
	}
}

func (app *App) Run(templatePath string) error {
	ctx, cancel := context.WithCancel(context.Background())
	app.listenForClose(cancel)

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

	app.file.Run(time.Millisecond * 100)
	app.cron.Start()
	app.pool.Run()
	processRunnerClosedChan := make(chan struct{})
	go func() {
		err := app.process.Run()
		if err != nil {
			close(processRunnerClosedChan)
		}

	}()

	select {
	case <-ctx.Done():
	case <-processRunnerClosedChan:
	}
	app.cron.Stop()
	app.file.Stop()
	app.pool.Stop()

	return nil
}

func (app *App) listenForClose(cancel context.CancelFunc) {
	go func() {
		sig := make(chan os.Signal)
		signal.Notify(sig, os.Interrupt)
		<-sig
		cancel()
	}()
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
		shell := executor.NewShell(app.context, app.logger)
		spec.Execute.Config.Decode(shell)
		svc.executor = shell
	}

	return svc
}
