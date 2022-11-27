package service

import (
	"github.com/mickyco94/saucisson/internal/dependencies"
	"github.com/robfig/cron/v3"
)

// Dependencies provides access to application wide resources
type Dependencies struct {
	Cron         *cron.Cron
	FileListener *dependencies.FileListener
}

//? Maybe too micro

func (deps *Dependencies) Start() {
	deps.Cron.Start()
	deps.FileListener.Run()
}

func (deps *Dependencies) Stop() {
	deps.FileListener.Stop()
	crnContext := deps.Cron.Stop()
	<-crnContext.Done()
}
