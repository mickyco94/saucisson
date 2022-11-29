package app

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/mickyco94/saucisson/internal/component"
	"github.com/mickyco94/saucisson/internal/parser"
	"github.com/mickyco94/saucisson/internal/service"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

//! Todo, just listening to ctx.Done() for the fuck of it in a lot of places
//? Totally unnecessary

type App struct {
	context      context.Context
	logger       logrus.FieldLogger
	cron         *cron.Cron
	filelistener *service.FileListener
	executorPool *ExecutorPool
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
		executorPool: NewExecutorPool(ctx, 5), //TODO: Drive from config
		logger:       logger,
		cron:         cron.New(cron.WithSeconds()),
		filelistener: service.NewFileListener(ctx, logger),
	}
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

	for _, s := range config.Services {
		svc := app.ConstructService(s)
		for _, fileCond := range svc.FileCondition {
			err := app.filelistener.HandleFunc(fileCond, func() {
				app.executorPool.Enqueue(svc.Executor)
			})

			if err != nil {
				panic(err)
			}
		}
		for _, cronCond := range svc.CronConditions {
			_, err := app.cron.AddFunc(cronCond.Schedule, func() {
				app.executorPool.Enqueue(svc.Executor)
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

//TODO: Need to decorate executor so that it runs on a shared
//Executor pool, specifics of that idk. Using an interface seems gross
//Passing a context object, maybe have a chan for each executor...?
//Dictionary of channels...?

//Channel for each service? Fan out to each executor?

//Execution context type?

// type ExecutionContext struct {
// 	conditionContext any
// 	serviceName      string
// }

type ExecutorPool struct {
	runningMu sync.Mutex
	wg        sync.WaitGroup
	ctx       context.Context

	size int
	//Internal queue for work
	//TODO: Send more context :)
	jobs chan component.Executor
}

func NewExecutorPool(context context.Context, size int) *ExecutorPool {
	return &ExecutorPool{
		ctx:       context,
		size:      size,
		wg:        sync.WaitGroup{},
		runningMu: sync.Mutex{},
		jobs:      make(chan component.Executor),
	}
}

func (pool *ExecutorPool) Stop() {
	close(pool.jobs)
	pool.wg.Wait()
}

func (pool *ExecutorPool) Run() {
	pool.wg.Add(pool.size)

	for i := 0; i < pool.size; i++ {
		go func() {
			defer pool.wg.Done()

			select {
			case <-pool.ctx.Done():
				return
			case j, open := <-pool.jobs:
				if !open {
					return
				}
				j.Execute()
			}
		}()
	}
}

// Enqueue adds the execution to the queue
func (pool *ExecutorPool) Enqueue(xc component.Executor) {
	pool.jobs <- xc
}

// Service can have any number of conditions of different types
// That all need to be registered
// For all of those conditions, each executor needs to be registered
type Service struct {
	CronConditions []*component.CronCondition
	FileCondition  []*component.File
	Executor       component.Executor
}

// ConstructService constructs an actual implementation of a Service from
// a specification
func (app *App) ConstructService(spec parser.ServiceSpec) *Service {
	svc := &Service{
		CronConditions: make([]*component.CronCondition, 0),
		FileCondition:  make([]*component.File, 0),
		Executor:       nil,
	}

	if spec.Condition.Type == "cron" {
		cronCondition := &component.CronCondition{}
		cronCondition.Configure(spec.Condition.Config)
		svc.CronConditions = append(svc.CronConditions, cronCondition)
	} else if spec.Condition.Type == "file" {
		fileCondition := &component.File{}
		fileCondition.Configure(spec.Condition.Config)
		svc.FileCondition = append(svc.FileCondition, fileCondition)
	}

	if spec.Execute.Type == "shell" {
		shell := component.NewShell(app.context, app.logger)
		shell.Configure(spec.Execute.Config)
		svc.Executor = shell
	}

	return svc
}
