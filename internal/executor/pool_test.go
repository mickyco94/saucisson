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

	cancelledCtx, _ := context.WithTimeout(context.Background(), time.Millisecond)

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
