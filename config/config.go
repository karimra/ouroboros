package config

import (
	"encoding/json"
	"os"

	"github.com/adrg/xdg"
	_ "github.com/karimra/ouroboros/actions/all"
	_ "github.com/karimra/ouroboros/outputs/all"
	_ "github.com/karimra/ouroboros/processors/all"
	_ "github.com/karimra/ouroboros/triggers/all"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	configName   = ".orbrs"
	configModule = "config"
	envPrefix    = "ORBRS"
)

type Config struct {
	FileConfig *viper.Viper `mapstructure:"-" json:"-"`

	Flags      *Flags                            `json:"flags,omitempty"`
	Triggers   map[string]map[string]interface{} `mapstructure:"triggers,omitempty" json:"triggers,omitempty"`
	Actions    map[string]map[string]interface{} `mapstructure:"actions,omitempty" json:"actions,omitempty"`
	Outputs    map[string]map[string]interface{} `mapstructure:"outputs,omitempty" json:"outputs,omitempty"`
	Processors map[string]map[string]interface{} `mapstructure:"processors,omitempty" json:"processors,omitempty"`

	logger *log.Entry
}

type Flags struct {
	Config  string `mapstructure:"config,omitempty" json:"config,omitempty"`
	LogFile string `mapstructure:"log-file,omitempty" json:"log-file,omitempty"`
	Debug   bool   `mapstructure:"debug,omitempty" json:"debug,omitempty"`
}

func New() *Config {
	return &Config{
		FileConfig: viper.New(),
		Flags:      new(Flags),
		Triggers:   make(map[string]map[string]interface{}),
		Actions:    make(map[string]map[string]interface{}),
		Outputs:    make(map[string]map[string]interface{}),
		Processors: make(map[string]map[string]interface{}),
		//logger:     log.StandardLogger(),
	}
}

func (c *Config) Load() error {
	c.FileConfig.AutomaticEnv()
	if c.Flags.Config != "" {
		c.FileConfig.SetConfigFile(c.Flags.Config)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			return err
		}
		c.FileConfig.AddConfigPath(".")
		c.FileConfig.AddConfigPath(home)
		c.FileConfig.AddConfigPath(xdg.ConfigHome)
		c.FileConfig.AddConfigPath(xdg.ConfigHome + "/orbrs")
		c.FileConfig.SetConfigName(configName)
	}

	err := c.FileConfig.ReadInConfig()
	if err != nil {
		return err
	}

	err = c.FileConfig.Unmarshal(c)
	if err != nil {
		return err
	}
	err = c.validate()
	if err != nil {
		return err
	}
	c.setLogger()
	//
	// TODO: add ENV VARS expansion
	//
	b, _ := json.MarshalIndent(c, "", " ")
	c.logger.Debugf("read config:\n %s", string(b))
	return nil
}

func (c *Config) validate() error {
	// TODO:
	return nil
}

func (c *Config) setLogger() {
	stdLogger := log.StandardLogger()
	if c.Flags.LogFile != "" {
		f, err := os.OpenFile(c.Flags.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return
		}
		stdLogger.SetOutput(f)
		c.logger = stdLogger.WithField("module", configModule)

	} else if c.Flags.Debug || c.FileConfig.GetBool("debug") {
		stdLogger.SetLevel(log.DebugLevel)
		stdLogger.SetOutput(os.Stderr)
		c.logger = stdLogger.WithField("module", configModule)
	}
}
