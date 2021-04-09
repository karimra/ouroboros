package snmp_action

import (
	"context"

	"github.com/karimra/ouroboros/actions"
	log "github.com/sirupsen/logrus"
)

const (
	actionType = "snmp"
)

func init() {
	actions.Register(actionType, func() actions.Action {
		return &snmpAction{
			cfg:    new(cfg),
			logger: log.StandardLogger(),
		}
	})
}

type snmpAction struct {
	cfg    *cfg
	logger *log.Logger
	name   string
}

type cfg struct{}

func (a *snmpAction) Init(string, interface{}, ...actions.Option) error { return nil }
func (a *snmpAction) Do(context.Context, interface{}, map[string]interface{}) (interface{}, error) {
	return nil, nil
}
func (a *snmpAction) Name() string { return a.name }
func (a *snmpAction) WithLogger(logger *log.Logger) {
	if a.logger == nil {
		a.logger = logger
	}
}
func (a *snmpAction) WithProcessors(map[string]map[string]interface{}) {}
func (a *snmpAction) WithOutputs(map[string]map[string]interface{})    {}
