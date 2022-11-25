package condition

type Condition interface {
	Register(chan<- struct{})
}
