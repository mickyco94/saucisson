package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestStartStopMultiThread(t *testing.T) {
	pool := NewPool(logrus.New(), DefaultPoolSize)

	pool.Start()

	pool.Stop(context.Background())

	assert.False(t, pool.running)
}

func TestStopCancelledContext(t *testing.T) {
	pool := NewPool(logrus.New(), DefaultPoolSize)

	pool.Start()

	cancelledCtx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	<-time.After(5 * time.Millisecond)
	err := pool.Stop(cancelledCtx)

	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestStopRunningExecutors(t *testing.T) {
	pool := NewPool(logrus.New(), 1)

	pool.Start()

	done := false

	pool.Enqueue(Job{
		Service: "test",
		Executor: func(ctx context.Context) error {
			time.Sleep(500 * time.Millisecond)
			done = true
			return nil
		},
	})

	timeout, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	pool.Stop(timeout)

	assert.True(t, done)
}

func TestExecutorCanError(t *testing.T) {
	pool := NewPool(logrus.New(), 1)

	pool.Start()

	pool.Enqueue(Job{
		Service: "test",
		Executor: func(ctx context.Context) error {
			return errors.New("Woopsie")
		},
	})

	assert.True(t, pool.running)
}

func TestExecutorRecover(t *testing.T) {
	pool := NewPool(logrus.New(), 1)

	pool.Start()

	pool.Enqueue(Job{
		Service: "panic",
		Executor: func(ctx context.Context) error {
			panic("Panicking!")
		},
	})

	//Superfluous assertion, we're just checking the panic is not propagated
	//No way to assert that there are n go-routines still running
	assert.True(t, pool.running)
}
