package http_action

import (
	"context"

	"github.com/karimra/ouroboros/actions"
	log "github.com/sirupsen/logrus"
)

const (
	actionType = "http"
)

func init() {
	actions.Register(actionType, func() actions.Action {
		return &httpAction{
			cfg:    new(cfg),
			logger: log.StandardLogger(),
		}
	})
}

type httpAction struct {
	cfg    *cfg
	logger *log.Logger
	name   string
}

type cfg struct{}

func (a *httpAction) Init(string, interface{}, ...actions.Option) error { return nil }
func (a *httpAction) Do(context.Context, interface{}, map[string]interface{}) (interface{}, error) {
	return nil, nil
}
func (a *httpAction) Name() string { return a.name }
func (a *httpAction) WithLogger(logger *log.Logger) {
	if a.logger == nil {
		a.logger = logger
	}
}
func (a *httpAction) WithProcessors(map[string]map[string]interface{}) {}
func (a *httpAction) WithOutputs(map[string]map[string]interface{})    {}
