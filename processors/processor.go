package processors

import (
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
)

type Processor interface {
	Init(interface{}, ...Option) error
	Apply(interface{}) (interface{}, error)
	WithLogger(*log.Logger)
}

type Initializer func() Processor

var Processors = map[string]Initializer{}

func Register(name string, initFn Initializer) {
	Processors[name] = initFn
}

type Option func(Processor)

func WithLogger(l *log.Logger) Option {
	return func(p Processor) {
		p.WithLogger(l)
	}
}

func CreateProcessor(cfg map[string]interface{}) (Processor, error) {
	if aType, ok := cfg["type"]; ok {
		if pin, ok := Processors[aType.(string)]; ok {
			return pin(), nil
		}
		return nil, fmt.Errorf("unknown processor type %q", aType)
	}
	return nil, errors.New("missing type field under processor")
}
