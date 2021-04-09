package null_action

import (
	"context"

	"github.com/karimra/ouroboros/actions"
	log "github.com/sirupsen/logrus"
)

const (
	actionType = "nc"
)

func init() {
	actions.Register(actionType, func() actions.Action {
		return &ncAction{
			cfg:    new(cfg),
			logger: log.StandardLogger(),
		}
	})
}

type ncAction struct {
	cfg    *cfg
	logger *log.Logger
	name   string
}

type cfg struct{}

func (a *ncAction) Init(string, interface{}, ...actions.Option) error { return nil }
func (a *ncAction) Do(context.Context, interface{}, map[string]interface{}) (interface{}, error) {
	return nil, nil
}
func (a *ncAction) Name() string { return a.name }
func (a *ncAction) WithLogger(logger *log.Logger) {
	if a.logger == nil {
		a.logger = logger
	}
}
func (a *ncAction) WithProcessors(map[string]map[string]interface{}) {}
func (a *ncAction) WithOutputs(map[string]map[string]interface{})    {}
