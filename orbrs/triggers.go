package orbrs

import "github.com/karimra/ouroboros/triggers"

func (a *App) startTriggers() {
	for name, cfg := range a.Config.Triggers {
		a.logger.Infof("starting trigger %q", name)
		if oType, ok := cfg["type"]; ok {
			if in, ok := triggers.Triggers[oType.(string)]; ok {
				trigger := in()
				go a.startTrigger(name, cfg, trigger)
				return
			}
			a.logger.Infof("unknown trigger type %q", oType)
			return
		}
		a.logger.Infof("missing trigger type under %q", name)
	}
}

func (a *App) startTrigger(name string, cfg interface{}, trigger triggers.Trigger) {
	err := trigger.Start(a.ctx, cfg,
		triggers.WithLogger(a.logger),
		triggers.WithOutputs(a.ctx, a.Config.Outputs, a.Config.Processors, a.logger),
		triggers.WithActions(a.Config.Actions, a.Config.Processors, a.Config.Outputs, a.logger),
		triggers.WithProcessors(a.Config.Processors, a.logger),
	)
	if err != nil {
		a.logger.Errorf("failed to init trigger %q: %v", name, err)
		return
	}
	a.m.Lock()
	a.triggers[name] = trigger
	a.m.Unlock()
}
