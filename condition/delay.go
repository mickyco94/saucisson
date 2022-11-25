package condition

import "time"

type MockCondition struct {
	SleepTime time.Duration
}

func (m *MockCondition) Register(trigger chan<- struct{}) {
	go func() {
		for {
			time.Sleep(m.SleepTime)
			trigger <- struct{}{}
		}
	}()
}
