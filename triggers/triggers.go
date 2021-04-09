package triggers

import (
	"context"

	log "github.com/sirupsen/logrus"
)

type Trigger interface {
	Start(context.Context, interface{}, ...Option) error

	WithActions(map[string]map[string]interface{}, map[string]map[string]interface{}, map[string]map[string]interface{}, *log.Logger)
	WithLogger(*log.Logger)
	WithProcessors(map[string]map[string]interface{}, *log.Logger)
	WithOutputs(context.Context, map[string]map[string]interface{}, map[string]map[string]interface{}, *log.Logger)

	Close() error
}

type Initializer func() Trigger

var Triggers = map[string]Initializer{}

func Register(name string, initFn Initializer) {
	Triggers[name] = initFn
}

type Option func(Trigger)

func WithActions(acts, procs, outs map[string]map[string]interface{}, l *log.Logger) Option {
	return func(i Trigger) {
		i.WithActions(acts, procs, outs, l)
	}
}

func WithLogger(l *log.Logger) Option {
	return func(i Trigger) {
		i.WithLogger(l)
	}
}

func WithProcessors(procs map[string]map[string]interface{}, l *log.Logger) Option {
	return func(i Trigger) {
		i.WithProcessors(procs, l)
	}
}

func WithOutputs(ctx context.Context, outs, procs map[string]map[string]interface{}, l *log.Logger) Option {
	return func(i Trigger) {
		i.WithOutputs(ctx, outs, procs, l)
	}
}
