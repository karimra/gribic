package config

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/karimra/gnmic/utils"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	configName = ".gribic"
	envPrefix  = "GRIBIC"
)

type Config struct {
	GlobalFlags `mapstructure:",squash"`
	LocalFlags  `mapstructure:",squash"`
	FileConfig  *viper.Viper `mapstructure:"-" json:"-" yaml:"-" `

	GnmiServer *gnmiServer `mapstructure:"gnmi-server,omitempty" json:"gnmi-server,omitempty" yaml:"gnmi-server,omitempty"`
	logger     *log.Entry
}

type GlobalFlags struct {
	CfgFile       string
	Address       []string      `mapstructure:"address,omitempty" json:"address,omitempty" yaml:"address,omitempty"`
	Username      string        `mapstructure:"username,omitempty" json:"username,omitempty" yaml:"username,omitempty"`
	Password      string        `mapstructure:"password,omitempty" json:"password,omitempty" yaml:"password,omitempty"`
	Port          string        `mapstructure:"port,omitempty" json:"port,omitempty" yaml:"port,omitempty"`
	Insecure      bool          `mapstructure:"insecure,omitempty" json:"insecure,omitempty" yaml:"insecure,omitempty"`
	TLSCa         string        `mapstructure:"tls-ca,omitempty" json:"tls-ca,omitempty" yaml:"tls-ca,omitempty"`
	TLSCert       string        `mapstructure:"tls-cert,omitempty" json:"tls-cert,omitempty" yaml:"tls-cert,omitempty"`
	TLSKey        string        `mapstructure:"tls-key,omitempty" json:"tls-key,omitempty" yaml:"tls-key,omitempty"`
	TLSMinVersion string        `mapstructure:"tls-min-version,omitempty" json:"tls-min-version,omitempty" yaml:"tls-min-version,omitempty"`
	TLSMaxVersion string        `mapstructure:"tls-max-version,omitempty" json:"tls-max-version,omitempty" yaml:"tls-max-version,omitempty"`
	TLSVersion    string        `mapstructure:"tls-version,omitempty" json:"tls-version,omitempty" yaml:"tls-version,omitempty"`
	Timeout       time.Duration `mapstructure:"timeout,omitempty" json:"timeout,omitempty" yaml:"timeout,omitempty"`
	SkipVerify    bool          `mapstructure:"skip-verify,omitempty" json:"skip-verify,omitempty" yaml:"skip-verify,omitempty"`
	ProxyFromEnv  bool          `mapstructure:"proxy-from-env,omitempty" json:"proxy-from-env,omitempty" yaml:"proxy-from-env,omitempty"`
	Gzip          bool          `mapstructure:"gzip,omitempty" json:"gzip,omitempty" yaml:"gzip,omitempty"`
	Format        string        `mapstructure:"format,omitempty" json:"format,omitempty" yaml:"format,omitempty"`
	Debug         bool          `mapstructure:"debug,omitempty" json:"debug,omitempty" yaml:"debug,omitempty"`
	//
	ElectionID string `mapstructure:"election-id,omitempty" json:"election-id,omitempty" yaml:"election-id,omitempty"`
}

type LocalFlags struct {
	// Get
	GetNetworkInstance    string
	GetAFT                string
	GetNetworkInstanceAll bool

	// flush
	FlushNetworkInstance    string
	FlushNetworkInstanceAll bool
	FlushElectionIDOverride bool

	// modify redundancy
	// ModifySessionRedundancyAllPrimary    bool
	ModifySessionRedundancySinglePrimary bool
	// modify persistence
	ModifySessionPersistancePreserve bool
	// modify ack
	// ModifySessionRibAck    bool
	ModifySessionRibFibAck bool
	// modify operations
	ModifyInputFile string
}

func New() *Config {
	return &Config{
		GlobalFlags{},
		LocalFlags{},
		viper.NewWithOptions(viper.KeyDelimiter("/")),
		nil,
		nil,
	}
}

func (c *Config) SetLogger() {
	logger := log.StandardLogger()
	if c.Debug {
		logger.SetLevel(log.DebugLevel)
	}
	c.logger = log.NewEntry(logger)
}

func (c *Config) Load(ctx context.Context) error {
	c.FileConfig.SetEnvPrefix(envPrefix)
	c.FileConfig.SetEnvKeyReplacer(strings.NewReplacer("/", "_", "-", "_"))
	c.FileConfig.AutomaticEnv()
	if c.GlobalFlags.CfgFile != "" {
		c.FileConfig.SetConfigFile(c.GlobalFlags.CfgFile)
		configBytes, err := utils.ReadFile(ctx, c.FileConfig.ConfigFileUsed())
		if err != nil {
			return err
		}
		err = c.FileConfig.ReadConfig(bytes.NewBuffer(configBytes))
		if err != nil {
			return err
		}
	} else {
		home, err := homedir.Dir()
		if err != nil {
			return err
		}
		c.FileConfig.AddConfigPath(".")
		c.FileConfig.AddConfigPath(home)
		c.FileConfig.AddConfigPath(xdg.ConfigHome)
		c.FileConfig.AddConfigPath(xdg.ConfigHome + "/gnoic")
		c.FileConfig.SetConfigName(configName)
	}

	err := c.FileConfig.ReadInConfig()
	if err != nil {
		return err
	}

	err = c.FileConfig.Unmarshal(c.FileConfig)
	if err != nil {
		return err
	}
	// c.mergeEnvVars()
	// return c.expandOSPathFlagValues()
	return nil
}

func (c *Config) SetPersistantFlagsFromFile(cmd *cobra.Command) {
	// set debug and log values from file before other persistant flags
	cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if f.Name == "debug" || f.Name == "log" {
			if !f.Changed && c.FileConfig.IsSet(f.Name) {
				c.setFlagValue(cmd, f.Name, c.FileConfig.Get(f.Name))
			}
		}
	})
	//
	cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if f.Name == "debug" || f.Name == "log" {
			return
		}
		if c.Debug {
			c.logger.Printf("cmd=%s, flagName=%s, changed=%v, isSetInFile=%v",
				cmd.Name(), f.Name, f.Changed, c.FileConfig.IsSet(f.Name))
		}
		if !f.Changed && c.FileConfig.IsSet(f.Name) {
			c.setFlagValue(cmd, f.Name, c.FileConfig.Get(f.Name))
		}
	})
}

func (c *Config) SetLocalFlagsFromFile(cmd *cobra.Command) {
	cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
		flagName := fmt.Sprintf("%s-%s", cmd.Name(), f.Name)
		if c.Debug {
			c.logger.Printf("cmd=%s, flagName=%s, changed=%v, isSetInFile=%v",
				cmd.Name(), f.Name, f.Changed, c.FileConfig.IsSet(flagName))
		}
		if !f.Changed && c.FileConfig.IsSet(flagName) {
			c.setFlagValue(cmd, f.Name, c.FileConfig.Get(flagName))
		}
	})
}

func (c *Config) setFlagValue(cmd *cobra.Command, fName string, val interface{}) {
	switch val := val.(type) {
	case []interface{}:
		if c.Debug {
			c.logger.Printf("cmd=%s, flagName=%s, valueType=%T, length=%d, value=%#v",
				cmd.Name(), fName, val, len(val), val)
		}
		nVal := make([]string, 0, len(val))
		for _, v := range val {
			nVal = append(nVal, fmt.Sprintf("%v", v))
		}
		cmd.Flags().Set(fName, strings.Join(nVal, ","))
	default:
		if c.Debug {
			c.logger.Printf("cmd=%s, flagName=%s, valueType=%T, value=%#v",
				cmd.Name(), fName, val, val)
		}
		cmd.Flags().Set(fName, fmt.Sprintf("%v", val))
	}
}
