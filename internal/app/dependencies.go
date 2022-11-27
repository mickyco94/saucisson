package app

import (
	"github.com/mickyco94/saucisson/internal/condition"
	"github.com/robfig/cron/v3"
)

// Dependencies provides access to application wide resources
type Dependencies struct {
	cron         *cron.Cron
	fileListener *condition.FileListener
}

//? Maybe too micro

func (deps *Dependencies) Start() {
	deps.cron.Start()
	deps.fileListener.Run()
}

func (deps *Dependencies) Stop() {
	deps.fileListener.Stop()
	crnContext := deps.cron.Stop()
	<-crnContext.Done()
}
