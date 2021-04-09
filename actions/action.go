package actions

import (
	"context"
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
)

type Action interface {
	Init(string, interface{}, ...Option) error
	Name() string
	Do(context.Context, interface{}, map[string]interface{}) (interface{}, error)

	WithLogger(*log.Logger)
	WithProcessors(map[string]map[string]interface{})
	WithOutputs(map[string]map[string]interface{})
}

type Initializer func() Action

var Actions = map[string]Initializer{}

func Register(name string, initFn Initializer) {
	Actions[name] = initFn
}

type Option func(Action)

func WithLogger(l *log.Logger) Option {
	return func(i Action) {
		i.WithLogger(l)
	}
}

func WithProcessors(procs map[string]map[string]interface{}) Option {
	return func(i Action) {
		i.WithProcessors(procs)
	}
}

func WithOutputs(outs map[string]map[string]interface{}) Option {
	return func(i Action) {
		i.WithOutputs(outs)
	}
}

func CreateAction(cfg map[string]interface{}) (Action, error) {
	if aType, ok := cfg["type"]; ok {
		if ain, ok := Actions[aType.(string)]; ok {
			return ain(), nil
		}
		return nil, fmt.Errorf("unknown action type %q", aType)
	}
	return nil, errors.New("missing type field under action")
}
