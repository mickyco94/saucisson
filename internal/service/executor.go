package service

// An executor is an abstractions that represents
// some parameterless invocation. Executors are run
// when a `Condition` is satisfied.

type Executor interface {

	// Execute will run the wrapped function
	// All errors are logged to the Service diagnostics
	Execute() error
}
