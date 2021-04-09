package null_action

import (
	"context"

	"github.com/karimra/ouroboros/actions"
	log "github.com/sirupsen/logrus"
)

const (
	actionType = "ssh"
)

func init() {
	actions.Register(actionType, func() actions.Action {
		return &sshAction{
			cfg:    new(cfg),
			logger: log.StandardLogger(),
		}
	})
}

type sshAction struct {
	cfg    *cfg
	logger *log.Logger
	name   string
}

type cfg struct{}

func (a *sshAction) Init(string, interface{}, ...actions.Option) error { return nil }
func (a *sshAction) Do(context.Context, interface{}, map[string]interface{}) (interface{}, error) {
	return nil, nil
}
func (a *sshAction) Name() string { return a.name }
func (a *sshAction) WithLogger(logger *log.Logger) {
	if a.logger == nil {
		a.logger = logger
	}
}
func (a *sshAction) WithProcessors(map[string]map[string]interface{}) {}
func (a *sshAction) WithOutputs(map[string]map[string]interface{})    {}
