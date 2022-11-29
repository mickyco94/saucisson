package app

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/mickyco94/saucisson/internal/condition"
	"github.com/mickyco94/saucisson/internal/executor"
	"github.com/mickyco94/saucisson/internal/parser"
	"github.com/mickyco94/saucisson/internal/service"
	"github.com/sirupsen/logrus"
)

type App struct {
	context      context.Context
	logger       logrus.FieldLogger
	cron         *service.Cron
	filelistener *service.FileListener
	executorPool *executor.Pool
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
		executorPool: executor.NewExecutorPool(logger, 5), //TODO: Drive from config
		logger:       logger,
		cron:         service.NewCron(),
		filelistener: service.NewFileListener(ctx, logger),
	}
}

func (app *App) Run(templatePath string) error {
	file, err := os.Open(templatePath)
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

	for _, s := range config.Services {
		svc := app.construct(s)
		for _, fileCond := range svc.files {
			err := app.filelistener.HandleFunc(fileCond, func() {
				app.executorPool.Enqueue(executor.Job{
					Service:  s.Name,
					Executor: svc.executor,
				})
			})

			if err != nil {
				panic(err)
			}
		}
		for _, cronCond := range svc.crons {
			app.cron.HandleFunc(cronCond, func() {
				app.executorPool.Enqueue(executor.Job{
					Service:  s.Name,
					Executor: svc.executor,
				})
			})
			if err != nil {
				panic(err)
			}
		}
	}

	//Start producers
	app.filelistener.Run(time.Millisecond * 100)
	app.cron.Start()
	app.executorPool.Run()

	//Listen for cancellation
	//Should be a select on multiple things really
	<-ctx.Done()

	app.cron.Stop()
	app.filelistener.Stop()
	app.executorPool.Stop()

	return nil
}

// definition can have any number of conditions of different types
// That all need to be registered
// For all of those conditions, each executor needs to be registered
type definition struct {
	crons    []*condition.Cron
	files    []*condition.File
	executor executor.Executor
}

// construct constructs an actual implementation of a Service from
// a specification
func (app *App) construct(spec parser.ServiceSpec) *definition {
	svc := &definition{
		crons:    make([]*condition.Cron, 0),
		files:    make([]*condition.File, 0),
		executor: nil,
	}

	switch spec.Condition.Type {
	case condition.CronKey:
		cronCondition := &condition.Cron{}
		spec.Condition.Config.Decode(cronCondition)
		svc.crons = append(svc.crons, cronCondition)
	case condition.FileKey:
		fileCondition := &condition.File{}
		spec.Condition.Config.Decode(fileCondition)
		svc.files = append(svc.files, fileCondition)
	}

	switch spec.Execute.Type {
	case "shell":
		shell := executor.NewShell(app.context, app.logger)
		shell.Configure(spec.Execute.Config)
		svc.executor = shell
	}

	return svc
}
