package registry

import (
	"errors"

	"github.com/mickyco94/saucisson/internal/component"
	"github.com/mickyco94/saucisson/internal/dependencies"
	"github.com/mickyco94/saucisson/internal/parser"
)

func (r *Registry) RegisterCondition(name string, f ConditionFactoryFunc) error {
	_, exists := r.conditions[name]
	if exists {
		return errors.New("Existing condition already registered under that key")
	}
	r.conditions[name] = f
	return nil
}

func (r *Registry) RegisterExecutor(name string, f ExecutorFactoryFunc) error {
	_, exists := r.executors[name]
	if exists {
		return errors.New("Existing executor already registered under that key")
	}
	r.executors[name] = f
	return nil
}

func NewRegistry(deps *dependencies.Dependencies) *Registry {
	return &Registry{
		d:          deps,
		conditions: make(map[string]ConditionFactoryFunc),
		executors:  make(map[string]ExecutorFactoryFunc),
	}
}

type ConditionFactoryFunc func(c parser.Raw, r *dependencies.Dependencies) (component.Condition, error)
type ExecutorFactoryFunc func(c parser.Raw, r *dependencies.Dependencies) (component.Executor, error)

// Registry holds all the factories
type Registry struct {
	d          *dependencies.Dependencies
	conditions map[string]ConditionFactoryFunc
	executors  map[string]ExecutorFactoryFunc
}

func (r *Registry) ConditionFromConfig(conf parser.Raw) (component.Condition, error) {
	componentName, err := conf.Name()

	if err != nil {
		return nil, err
	}

	constructor, exists := r.conditions[componentName]

	if !exists {
		return nil, errors.New("Component undefined")
	}

	configSection, err := conf.ExtractSection(componentName)

	return constructor(configSection, r.d)
}

func (r *Registry) ExecutorFromConfig(conf parser.Raw) (component.Executor, error) {
	componentName, err := conf.Name()

	if err != nil {
		return nil, err
	}

	constructor, exists := r.executors[componentName]

	if !exists {
		return nil, errors.New("Component undefined")
	}

	configSection, err := conf.ExtractSection(componentName)

	return constructor(configSection, r.d)
}
