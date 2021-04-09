package jq_proc

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/itchyny/gojq"
	"github.com/karimra/ouroboros/processors"
	"github.com/karimra/ouroboros/utils"
	log "github.com/sirupsen/logrus"
)

const (
	processorName     = "jq"
	defaultCondition  = "any([true])"
	defaultExpression = "."
)

func init() {
	processors.Register(processorName, func() processors.Processor {
		return &jqProc{
			cfg: new(cfg),
		}
	})
}

type jqProc struct {
	cfg *cfg

	cond *gojq.Code
	expr *gojq.Code

	logger *log.Entry
}

type cfg struct {
	Condition  string
	Expression string
}

func (p *jqProc) Init(cfg interface{}, opts ...processors.Option) error {
	err := utils.DecodeConfig(cfg, p.cfg)
	if err != nil {
		return err
	}
	for _, opt := range opts {
		opt(p)
	}

	err = p.setDefaults()
	if err != nil {
		return err
	}

	p.cfg.Condition = strings.TrimSpace(p.cfg.Condition)
	q, err := gojq.Parse(p.cfg.Condition)
	if err != nil {
		return err
	}
	p.cond, err = gojq.Compile(q)
	if err != nil {
		return err
	}

	p.cfg.Expression = strings.TrimSpace(p.cfg.Expression)
	q, err = gojq.Parse(p.cfg.Expression)
	if err != nil {
		return err
	}
	p.expr, err = gojq.Compile(q)
	if err != nil {
		return err
	}
	p.logger.Infof("starting processor %+v", p.cfg)
	return nil
}

func (p *jqProc) Apply(in interface{}) (interface{}, error) {
	var jin interface{}
	switch in := in.(type) {
	case []uint8:
		err := json.Unmarshal(in, &jin)
		if err != nil {
			return nil, err
		}
		p.logger.Debugf("input object: (%T)%v", jin, jin)
	default:
		p.logger.Debugf("input object: (%T)%v", jin, jin)
		jin = in
	}
	iter := p.cond.Run(jin)
	res, ok := iter.Next()
	if !ok {
		// iterator not done, so the final result won't be a boolean
		return jin, nil
	}
	if err, ok := res.(error); ok {
		p.logger.Debugf("jq condition failed: %v", err)
		return nil, err
	}
	p.logger.Debugf("jq condition result: (%T)%v", res, res)
	switch res := res.(type) {
	case bool:
		if res {
			// apply expression
			iter := p.expr.Run(jin)
			var res []interface{}
			for {
				r, ok := iter.Next()
				if !ok {
					break
				}
				switch r := r.(type) {
				case error:
					p.logger.Errorf("jq processor expression failed: %v", r)
					return nil, r
				default:
					res = append(res, r)
				}
			}
			if len(res) == 1 {
				return res[0], nil
			}
			return res, nil
		}
		return jin, nil
	default:
		p.logger.Errorf("unexpected condition type %T", res)
		return nil, fmt.Errorf("unexpected condition type %T", res)
	}
}

func (p *jqProc) WithLogger(logger *log.Logger) {
	if p.logger == nil {
		p.logger = logger.WithField("plugin", "processor_"+processorName)
	}
}

func (p *jqProc) setDefaults() error {
	if p.cfg.Condition == "" {
		p.cfg.Condition = defaultCondition
	}
	if p.cfg.Expression == "" {
		p.cfg.Expression = defaultExpression
	}
	return nil
}
