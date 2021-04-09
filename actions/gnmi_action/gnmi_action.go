package gnmi_action

import (
	"context"

	"github.com/karimra/ouroboros/actions"
	log "github.com/sirupsen/logrus"
)

const (
	actionType = "gnmi"
)

func init() {
	actions.Register(actionType, func() actions.Action {
		return &gnmiAction{
			cfg:    new(cfg),
			logger: log.StandardLogger(),
		}
	})
}

type gnmiAction struct {
	cfg    *cfg
	logger *log.Logger
	name   string
}

type cfg struct{}

func (a *gnmiAction) Init(string, interface{}, ...actions.Option) error { return nil }
func (a *gnmiAction) Do(context.Context, interface{}, map[string]interface{}) (interface{}, error) {
	return nil, nil
}
func (a *gnmiAction) Name() string { return a.name }
func (a *gnmiAction) WithLogger(logger *log.Logger) {
	if a.logger == nil {
		a.logger = logger
	}
}
func (a *gnmiAction) WithProcessors(map[string]map[string]interface{}) {}
func (a *gnmiAction) WithOutputs(map[string]map[string]interface{})    {}
