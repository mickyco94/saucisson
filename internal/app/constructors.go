package app

import (
	"errors"

	"github.com/mickyco94/saucisson/internal/component"
	"github.com/mickyco94/saucisson/internal/parser"
	filewatcher "github.com/radovskyb/watcher"
)

func FileListenerFactory(s *services, c parser.Raw) (component.Condition, error) {

	path, err := c.ExtractString("path")
	if err != nil {
		return nil, err
	}

	op, err := c.ExtractString("operation")

	if err != nil {
		return nil, err
	}

	var actual filewatcher.Op
	switch op {
	case "create":
		actual = filewatcher.Create
	case "rename":
		actual = filewatcher.Rename
	case "delete":
		actual = filewatcher.Remove
	case "chmod":
		actual = filewatcher.Chmod
	case "update":
		actual = filewatcher.Write
	default:
		return nil, errors.New("Unsupported op")
	}

	return component.NewFile(path, actual, s.filelistener), nil
}

func CronConditionFactory(s *services, c parser.Raw) (component.Condition, error) {

	schedule, err := c.ExtractString("schedule")
	if err != nil {
		return nil, err
	}

	return component.NewCronCondition(schedule, s.cron), nil
}

func ShellExecutorFactory(s *services, c parser.Raw) (component.Executor, error) {

	command, err := c.ExtractString("command")
	if err != nil {
		return nil, err
	}

	return component.NewShell(command), nil
}

type ExecutorConstructor func(s *services, c parser.Raw) (component.Executor, error)
type ConditionConstructor func(s *services, c parser.Raw) (component.Condition, error)

var ExecutorConstructors map[string]ExecutorConstructor = map[string]ExecutorConstructor{
	"shell": ShellExecutorFactory,
}

var ConditionConstructors map[string]ConditionConstructor = map[string]ConditionConstructor{
	"cron": CronConditionFactory,
	"file": FileListenerFactory,
}
