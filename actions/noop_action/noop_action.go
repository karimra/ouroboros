package noop_action

import (
	"context"

	"github.com/karimra/ouroboros/actions"
	"github.com/karimra/ouroboros/utils"
	log "github.com/sirupsen/logrus"
)

const (
	actionType = "noop"
)

func init() {
	actions.Register(actionType, func() actions.Action {
		return &noopAction{
			Cfg: new(Config),
		}
	})
}

type noopAction struct {
	Cfg    *Config
	name   string
	logger *log.Entry
}

type Config struct {
	Outputs []string
}

func (a *noopAction) Init(name string, cfg interface{}, opts ...actions.Option) error {
	err := utils.DecodeConfig(cfg, a.Cfg)
	if err != nil {
		return err
	}
	a.name = name
	for _, opt := range opts {
		opt(a)
	}
	a.logger.Infof("initalized action %+v", a)
	return nil
}

func (a *noopAction) Do(ctx context.Context, d interface{}, env map[string]interface{}) (interface{}, error) {
	return d, nil
}
func (a *noopAction) Name() string { return a.name }
func (a *noopAction) WithLogger(logger *log.Logger) {
	if a.logger == nil {
		a.logger = logger.WithField("plugin", "action_"+actionType)
	}
}
func (a *noopAction) WithProcessors(map[string]map[string]interface{}) {}
func (a *noopAction) WithOutputs(map[string]map[string]interface{})    {}
