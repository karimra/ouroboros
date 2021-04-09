package nats_output

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/karimra/ouroboros/outputs"
	"github.com/karimra/ouroboros/processors"
	"github.com/karimra/ouroboros/utils"
	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
)

const (
	outputName              = "nats"
	natsConnectWait         = 2 * time.Second
	natsReconnectBufferSize = 100 * 1024 * 1024
	defaultSubjectName      = "orbrs.events.out"
	defaultFormat           = "event"
	defaultNumWorkers       = 1
	defaultWriteTimeout     = 5 * time.Second
	defaultAddress          = "localhost:4222"
	loggingPrefix           = "nats_output"
)

func init() {
	outputs.Register(outputName, func() outputs.Output {
		return &NatsOutput{
			cfg: &cfg{},
			//logger: log.StandardLogger(),
			wg:      new(sync.WaitGroup),
			msgChan: make(chan interface{}),
		}
	})
}

type NatsOutput struct {
	cfg *cfg

	ctx     context.Context
	cfn     context.CancelFunc
	procs   []processors.Processor
	msgChan chan interface{}
	wg      *sync.WaitGroup
	logger  *log.Entry
}

type cfg struct {
	Name            string        `mapstructure:"name,omitempty"`
	Address         string        `mapstructure:"address,omitempty"`
	Subject         string        `mapstructure:"subject,omitempty"`
	Username        string        `mapstructure:"username,omitempty"`
	Password        string        `mapstructure:"password,omitempty"`
	ConnectTimeWait time.Duration `mapstructure:"connect-time-wait,omitempty"`
	Debug           bool          `mapstructure:"debug,omitempty"`
	NumWorkers      int           `mapstructure:"num-workers,omitempty"`
	BufferSize      int           `mapstructure:"buffer-size,omitempty"`
	WriteTimeout    time.Duration `mapstructure:"write-timeout,omitempty"`
	Processors      []string      `mapstructure:"processors,omitempty"`
}

func (n *NatsOutput) Init(ctx context.Context, cfg interface{}, opts ...outputs.Option) error {
	err := utils.DecodeConfig(cfg, n.cfg)
	if err != nil {
		return err
	}
	for _, opt := range opts {
		opt(n)
	}
	err = n.setDefaults()
	if err != nil {
		return err
	}
	n.ctx, n.cfn = context.WithCancel(ctx)
	n.logger.Infof("output starting with config: %+v", n.cfg)
	n.wg.Add(n.cfg.NumWorkers)
	for i := 0; i < n.cfg.NumWorkers; i++ {
		go n.worker(ctx, i)
	}
	return nil
}

func (n *NatsOutput) Write(ctx context.Context, d interface{}) error {
	n.logger.Debugf("data received of type %T", d)
	switch d := d.(type) {
	case nil:
		n.logger.Debug("nil data received, skipping...")
		return nil
	case []uint8:
		if len(d) == 0 {
			n.logger.Debug("nil data received, skipping...")
			return nil
		}
	case []interface{}:
		if len(d) == 0 {
			n.logger.Debug("nil data received, skipping...")
			return nil
		}
	case map[string]interface{}:
		if len(d) == 0 {
			n.logger.Debug("nil data received, skipping...")
			return nil
		}
	}
	n.logger.Infof("writing data to output: %v", d)
	tctx, cancel := context.WithTimeout(ctx, n.cfg.WriteTimeout)
	defer cancel()
	select {
	case <-tctx.Done():
		return tctx.Err()
	case n.msgChan <- d:
	}
	return nil
}

func (n *NatsOutput) Close() error { return nil }

func (n *NatsOutput) WithLogger(logger *log.Logger) {
	if n.logger == nil {
		n.logger = logger.WithField("plugin", loggingPrefix)
	}
}

func (n *NatsOutput) WithProcessors(procs map[string]map[string]interface{}, l *log.Logger) {
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

func (n *NatsOutput) setDefaults() error {
	if n.cfg.Address == "" {
		n.cfg.Address = defaultAddress
	}
	if n.cfg.ConnectTimeWait <= 0 {
		n.cfg.ConnectTimeWait = natsConnectWait
	}
	if n.cfg.Subject == "" {
		n.cfg.Subject = defaultSubjectName
	}
	if n.cfg.Name == "" {
		n.cfg.Name = "gnmic-" + uuid.New().String()
	}
	if n.cfg.NumWorkers <= 0 {
		n.cfg.NumWorkers = defaultNumWorkers
	}
	if n.cfg.WriteTimeout <= 0 {
		n.cfg.WriteTimeout = defaultWriteTimeout
	}
	return nil
}

func (n *NatsOutput) worker(ctx context.Context, idx int) {
	defer n.wg.Done()
	var natsConn *nats.Conn
	var err error
	workerLogPrefix := fmt.Sprintf("worker-%d", idx)
	n.logger.Infof("%s starting", workerLogPrefix)
	wcfg := *n.cfg
	wcfg.Name = fmt.Sprintf("%s-%d", wcfg.Name, idx)
CRCONN:
	natsConn, err = n.createNATSConn(&wcfg)
	if err != nil {
		n.logger.Errorf("%s failed to create connection: %v", workerLogPrefix, err)
		time.Sleep(n.cfg.ConnectTimeWait)
		goto CRCONN
	}
	defer natsConn.Close()
	n.logger.Infof("%s initialized nats producer: %+v", workerLogPrefix, wcfg)
OUTER:
	for {
		select {
		case <-ctx.Done():
			n.logger.Infof("%s flushing", workerLogPrefix)
			natsConn.FlushTimeout(time.Second)
			n.logger.Infof("%s shutting down", workerLogPrefix)
			return
		case msg := <-n.msgChan:
			var data interface{}
			var err error
			data = msg
			for _, p := range n.procs {
				n.logger.Infof("applying processor: %v", p)
				data, err = p.Apply(data)
				if err != nil {
					n.logger.Errorf("failed to apply processor: %v", err)
					continue OUTER
				}
			}
			b, err := n.toBytes(data)
			if err != nil {
				n.logger.Errorf("failed to marshal result: %v", err)
				continue
			}
			subject := n.subjectName(wcfg)
			n.logger.Infof("%s publish %s", workerLogPrefix, string(b))
			err = natsConn.Publish(subject, b)
			if err != nil {
				if n.cfg.Debug {
					n.logger.Printf("%s failed to write to nats subject '%s': %v", workerLogPrefix, subject, err)
				}
				natsConn.Close()
				time.Sleep(wcfg.ConnectTimeWait)
				goto CRCONN
			}
		}
	}
}

func (n *NatsOutput) createNATSConn(c *cfg) (*nats.Conn, error) {
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
func (n *NatsOutput) Dial(network, address string) (net.Conn, error) {
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

func (n *NatsOutput) subjectName(cfg cfg) string { return cfg.Subject }

func (n *NatsOutput) toBytes(i interface{}) ([]byte, error) {
	switch i := i.(type) {
	case []uint8:
		return i, nil
	default:
		b, err := json.Marshal(i)
		if err != nil {
			return nil, err
		}
		return b, nil
	}
}
