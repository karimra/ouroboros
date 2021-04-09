package outputs

import (
	"context"
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
)

type Output interface {
	Init(context.Context, interface{}, ...Option) error
	Write(context.Context, interface{}) error
	Close() error

	WithLogger(*log.Logger)
	WithProcessors(map[string]map[string]interface{}, *log.Logger)
}

type Initializer func() Output

var Outputs = map[string]Initializer{}

func Register(name string, initFn Initializer) {
	Outputs[name] = initFn
}

type Option func(Output)

func WithLogger(l *log.Logger) Option {
	return func(o Output) {
		o.WithLogger(l)
	}
}

func WithProcessors(procs map[string]map[string]interface{}, l *log.Logger) Option {
	return func(o Output) {
		o.WithProcessors(procs, l)
	}
}

func CreateOutput(cfg map[string]interface{}) (Output, error) {
	if aType, ok := cfg["type"]; ok {
		if oin, ok := Outputs[aType.(string)]; ok {
			return oin(), nil
		}
		return nil, fmt.Errorf("unknown output type %q", aType)
	}
	return nil, errors.New("missing type field under output")
}
