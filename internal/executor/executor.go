package executor

import (
	"context"
	"errors"
)

// An executor is an abstractions that represents
// some parameterless invocation. Executors are run
// when a `Condition` is satisfied.

type Executor interface {

	// Execute will run the wrapped function
	// All errors are logged to the Service diagnostics
	//
	// Context is used for cancellation of the running Executors
	Execute(context.Context) error
}

// ErrTimeoutExceeded is an err that indicates the configured timeout for the execution
// has been exceeded.
// Timeout for executors can be set by setting the "timeout" property
// in the executor specification. The units are in seconds for this field.
var ErrTimeoutExceeded = errors.New("Execution timeout exceeded")
