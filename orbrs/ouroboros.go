package orbrs

import (
	"context"
	"io"
	"sync"

	"github.com/karimra/ouroboros/actions"
	"github.com/karimra/ouroboros/config"
	"github.com/karimra/ouroboros/outputs"
	"github.com/karimra/ouroboros/processors"
	"github.com/karimra/ouroboros/triggers"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type App struct {
	ctx context.Context
	cfn context.CancelFunc

	Config  *config.Config
	RootCmd *cobra.Command

	m          *sync.Mutex
	triggers   map[string]triggers.Trigger
	actions    map[string]actions.Action
	processors map[string]processors.Processor
	outputs    map[string]outputs.Output

	logger *log.Logger
}

func New() *App {
	ctx, cancel := context.WithCancel(context.Background())
	return &App{
		ctx:        ctx,
		cfn:        cancel,
		Config:     config.New(),
		m:          new(sync.Mutex),
		triggers:   make(map[string]triggers.Trigger),
		actions:    make(map[string]actions.Action),
		processors: make(map[string]processors.Processor),
		outputs:    make(map[string]outputs.Output),
		RootCmd:    new(cobra.Command),
		logger:     log.StandardLogger(),
	}
}

func (a *App) InitFlags() {
	a.RootCmd.ResetFlags()
	a.RootCmd.PersistentFlags().StringVarP(&a.Config.Flags.Config, "config", "c", "", "config file")
	a.RootCmd.PersistentFlags().StringVarP(&a.Config.Flags.LogFile, "log-file", "l", "", "log file path")
	a.RootCmd.PersistentFlags().BoolVarP(&a.Config.Flags.Debug, "debug", "d", false, "debug mode")
	a.RootCmd.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		a.Config.FileConfig.BindPFlag(flag.Name, flag)
	})
}

func (a *App) Start() {
	a.logger.Infoln("starting orbrs...")
	a.startTriggers()
	<-a.ctx.Done()
}

func (a *App) SetLogOutput(f io.Writer) {
	a.logger.SetOutput(f)
}

func (a *App) SetLogLevel(l log.Level) {
	a.logger.SetLevel(l)
}
