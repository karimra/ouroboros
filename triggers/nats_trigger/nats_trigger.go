package nats_trigger

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/karimra/ouroboros/actions"
	"github.com/karimra/ouroboros/outputs"
	"github.com/karimra/ouroboros/processors"
	"github.com/karimra/ouroboros/triggers"
	"github.com/karimra/ouroboros/utils"
	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
)

const (
	triggerName             = "nats"
	loggingPrefix           = "nats_trigger"
	natsReconnectBufferSize = 100 * 1024 * 1024
	defaultAddress          = "localhost:4222"
	natsConnectWait         = 2 * time.Second
	defaultSubject          = "orbrs.events.in"
	defaultNumWorkers       = 1
	defaultBufferSize       = 100
)

func init() {
	triggers.Register(triggerName, func() triggers.Trigger {
		return &NatsTrigger{
			cfg: new(cfg),
			wg:  new(sync.WaitGroup),
		}
	})
}

// NatsTrigger //
type NatsTrigger struct {
	cfg    *cfg
	ctx    context.Context
	cfn    context.CancelFunc
	logger *log.Entry

	wg      *sync.WaitGroup
	procs   []processors.Processor
	actions []actions.Action
	outputs []outputs.Output
}

type cfg struct {
	Name            string        `mapstructure:"name,omitempty" json:"name,omitempty"`
	Address         string        `mapstructure:"address,omitempty" json:"address,omitempty"`
	Subject         string        `mapstructure:"subject,omitempty" json:"subject,omitempty"`
	Queue           string        `mapstructure:"queue,omitempty" json:"queue,omitempty"`
	Username        string        `mapstructure:"username,omitempty" json:"username,omitempty"`
	Password        string        `mapstructure:"password,omitempty" json:"password,omitempty"`
	ConnectTimeWait time.Duration `mapstructure:"connect-time-wait,omitempty" json:"connect-time-wait,omitempty"`
	Debug           bool          `mapstructure:"debug,omitempty" json:"debug,omitempty"`
	NumWorkers      int           `mapstructure:"num-workers,omitempty" json:"num-workers,omitempty"`
	BufferSize      int           `mapstructure:"buffer-size,omitempty" json:"buffer-size,omitempty"`
	Processors      []string      `mapstructure:"processors,omitempty" json:"processors,omitempty"`
	Actions         []string      `mapstructure:"actions,omitempty" json:"actions,omitempty"`
	Outputs         []string      `mapstructure:"outputs,omitempty" json:"outputs,omitempty"`
}

// Start //
func (n *NatsTrigger) Start(ctx context.Context, cfg interface{}, opts ...triggers.Option) error {
	err := utils.DecodeConfig(cfg, n.cfg)
	if err != nil {
		return err
	}
	err = n.setDefaults()
	if err != nil {
		return err
	}
	for _, opt := range opts {
		opt(n)
	}

	n.ctx, n.cfn = context.WithCancel(ctx)
	// start outputs
	n.logger.Infof("trigger starting with config: %+v", n.cfg)
	n.wg.Add(n.cfg.NumWorkers)
	for i := 0; i < n.cfg.NumWorkers; i++ {
		go n.worker(ctx, i)
	}
	return nil
}

func (n *NatsTrigger) worker(ctx context.Context, idx int) {
	var nc *nats.Conn
	var err error
	var msgChan chan *nats.Msg
	workerLogPrefix := fmt.Sprintf("worker-%d", idx)
	n.logger.Printf("%s starting", workerLogPrefix)
	wcfg := *n.cfg
	wcfg.Name = fmt.Sprintf("%s-%d", wcfg.Name, idx)
START:
	nc, err = n.createNATSConn(&wcfg)
	if err != nil {
		n.logger.Errorf("%s failed to create NATS connection: %v", workerLogPrefix, err)
		time.Sleep(n.cfg.ConnectTimeWait)
		goto START
	}
	defer nc.Close()
	msgChan = make(chan *nats.Msg, n.cfg.BufferSize)
	sub, err := nc.ChanQueueSubscribe(n.cfg.Subject, n.cfg.Queue, msgChan)
	if err != nil {
		n.logger.Errorf("%s failed to create NATS subscription: %v", workerLogPrefix, err)
		time.Sleep(n.cfg.ConnectTimeWait)
		nc.Close()
		goto START
	}
	defer close(msgChan)
	defer sub.Unsubscribe()
OUTER:
	for {
		select {
		case <-ctx.Done():
			return
		case m, ok := <-msgChan:
			n.logger.Debugf("received msg: %+v", m)
			if !ok {
				n.logger.Errorf("%s channel closed, retrying...", workerLogPrefix)
				time.Sleep(n.cfg.ConnectTimeWait)
				nc.Close()
				goto START
			}
			if len(m.Data) == 0 {
				continue
			}
			if n.cfg.Debug {
				n.logger.Debugf("received msg, subject=%s, queue=%s, len=%d, data=%s", m.Subject, m.Sub.Queue, len(m.Data), string(m.Data))
			}

			var data interface{}
			data = m.Data
			for _, p := range n.procs {
				data, err = p.Apply(data)
				if err != nil {
					n.logger.Errorf("failed to apply processor: %v", err)
					continue OUTER
				}
			}

			var rs interface{}
			var err error
			env := make(map[string]interface{})
			rs = data
			for _, a := range n.actions {
				n.logger.Infof("applying action: %+v", a)
				rs, err = a.Do(ctx, rs, env)
				if err != nil {
					n.logger.Printf("action failed: %v", err)
				}
				env[a.Name()] = rs
				n.logger.Infof("applied action %q: result: %v", a.Name(), rs)
				n.logger.Infof("action %q new trigger env: %+v", a.Name(), env)
			}
			for _, o := range n.outputs {
				n.logger.Infof("sending result to output: %v", o)
				err = o.Write(ctx, rs)
				if err != nil {
					n.logger.Errorf("failed to marshal actions result: %v", err)
					// TODO:
					continue
				}
			}
		}
	}
}

// Close //
func (n *NatsTrigger) Close() error {
	n.cfn()
	n.wg.Wait()
	return nil
}

// SetLogger //
func (n *NatsTrigger) WithLogger(logger *log.Logger) {
	if n.logger == nil {
		n.logger = logger.WithField("plugin", loggingPrefix)
	}
}

func (n *NatsTrigger) WithActions(acts, procs, outs map[string]map[string]interface{}, l *log.Logger) {
	for _, name := range n.cfg.Actions {
		if aCfg, ok := acts[name]; ok {
			n.logger.Infof("initializing action %q", name)
			a, err := actions.CreateAction(aCfg)
			if err != nil {
				n.logger.Errorf("failed to initialize action %q: %v", name, err)
				continue
			}
			err = a.Init(name, aCfg,
				actions.WithLogger(l),
				actions.WithProcessors(procs),
				actions.WithOutputs(outs),
			)
			if err != nil {
				n.logger.Errorf("failed to init action %q: %v", a.Name(), err)
				continue
			}
			n.actions = append(n.actions, a)
			continue
		}
		n.logger.Warnf("action %q not found", name)
	}
}

func (n *NatsTrigger) WithProcessors(procs map[string]map[string]interface{}, l *log.Logger) {
	for _, name := range n.cfg.Processors {
		if pCfg, ok := procs[name]; ok {
			n.logger.Infof("initializing processor %q", name)
			p, err := processors.CreateProcessor(pCfg)
			if err != nil {
				n.logger.Errorf("failed to initialize processor %q: %v", name, err)
				continue
			}
			p.Init(pCfg, processors.WithLogger(l))
			n.procs = append(n.procs, p)
			continue
		}
		n.logger.Warnf("processor %q not found", name)
	}
}

func (n *NatsTrigger) WithOutputs(ctx context.Context, outs map[string]map[string]interface{}, procs map[string]map[string]interface{}, l *log.Logger) {
	for _, name := range n.cfg.Outputs {
		if pCfg, ok := outs[name]; ok {
			n.logger.Infof("initializing output %q", name)
			p, err := outputs.CreateOutput(pCfg)
			if err != nil {
				n.logger.Errorf("failed to initialize output %q: %v", name, err)
				continue
			}
			p.Init(ctx, pCfg, outputs.WithLogger(l), outputs.WithProcessors(procs, l))
			n.outputs = append(n.outputs, p)
			continue
		}
		n.logger.Warnf("output %q not found", name)
	}
}

// func (n *NatsTrigger) SetName(name string) {
// 	sb := strings.Builder{}
// 	if name != "" {
// 		sb.WriteString(name)
// 		sb.WriteString("-")
// 	}
// 	sb.WriteString(n.Cfg.Name)
// 	sb.WriteString("-nats-sub")
// 	n.Cfg.Name = sb.String()
// }

// helper functions

func (n *NatsTrigger) setDefaults() error {
	if n.cfg.Name == "" {
		n.cfg.Name = "orbrs-" + uuid.New().String()
	}
	if n.cfg.Subject == "" {
		n.cfg.Subject = defaultSubject
	}
	if n.cfg.Address == "" {
		n.cfg.Address = defaultAddress
	}
	if n.cfg.ConnectTimeWait <= 0 {
		n.cfg.ConnectTimeWait = natsConnectWait
	}
	if n.cfg.Queue == "" {
		n.cfg.Queue = n.cfg.Name
	}
	if n.cfg.NumWorkers <= 0 {
		n.cfg.NumWorkers = defaultNumWorkers
	}
	if n.cfg.BufferSize <= 0 {
		n.cfg.BufferSize = defaultBufferSize
	}
	return nil
}

func (n *NatsTrigger) createNATSConn(c *cfg) (*nats.Conn, error) {
	opts := []nats.Option{
		nats.Name(c.Name),
		nats.SetCustomDialer(n),
		nats.ReconnectWait(n.cfg.ConnectTimeWait),
		nats.ReconnectBufSize(natsReconnectBufferSize),
		nats.ErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, err error) {
			n.logger.Printf("NATS error: %v", err)
		}),
		nats.DisconnectHandler(func(c *nats.Conn) {
			n.logger.Println("Disconnected from NATS")
		}),
		nats.ClosedHandler(func(c *nats.Conn) {
			n.logger.Println("NATS connection is closed")
		}),
	}
	if c.Username != "" && c.Password != "" {
		opts = append(opts, nats.UserInfo(c.Username, c.Password))
	}
	nc, err := nats.Connect(c.Address, opts...)
	if err != nil {
		return nil, err
	}
	return nc, nil
}

// Dial //
func (n *NatsTrigger) Dial(network, address string) (net.Conn, error) {
	ctx, cancel := context.WithCancel(n.ctx)
	defer cancel()

	for {
		n.logger.Printf("attempting to connect to %s", address)
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		select {
		case <-n.ctx.Done():
			return nil, n.ctx.Err()
		default:
			d := &net.Dialer{}
			if conn, err := d.DialContext(ctx, network, address); err == nil {
				n.logger.Printf("successfully connected to NATS server %s", address)
				return conn, nil
			}
			time.Sleep(n.cfg.ConnectTimeWait)
		}
	}
}
