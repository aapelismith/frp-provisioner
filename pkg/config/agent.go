package config

import (
	"errors"
	"fmt"
	"github.com/frp-sigs/frp-provisioner/pkg/log"
	"github.com/spf13/pflag"
)

// AgentConfiguration is the agent configuration.
type AgentConfiguration struct {
	// Log is the log options struct for zap logger
	Log *log.Options `json:"log,omitempty"`
}

// AddFlags adds flags for a specific configuration to the specified FlagSet
func (c *AgentConfiguration) AddFlags(fs *pflag.FlagSet) {
	c.Log.AddFlags(fs)
}

// SetDefaults sets the default values for a specific configuration.
func (c *AgentConfiguration) SetDefaults() {
	c.Log.SetDefaults()
}

// Validate validates a specific configuration.
func (c *AgentConfiguration) Validate() (errs error) {
	if err := c.Log.Validate(); err != nil {
		errs = errors.Join(errs, fmt.Errorf("invalid log config, got: '%w'", err))
	}
	return errs
}

// NewAgentConfiguration create AgentConfiguration
func NewAgentConfiguration() *AgentConfiguration {
	return &AgentConfiguration{
		Log: log.NewOptions(),
	}
}
