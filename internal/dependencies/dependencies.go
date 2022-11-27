package dependencies

import (
	"github.com/robfig/cron/v3"
)

//!! Not sure how I feel about this package existing...

// Dependencies provides access to application wide resources
type Dependencies struct {
	Cron         *cron.Cron
	FileListener *FileListener
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
