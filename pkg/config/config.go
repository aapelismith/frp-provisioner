package config

import (
	"github.com/spf13/pflag"
	"kunstack.com/pharos/pkg/client/clientset"
	"kunstack.com/pharos/pkg/controller/loadbalancer"
	"kunstack.com/pharos/pkg/log"
	"kunstack.com/pharos/pkg/safe"
	"kunstack.com/pharos/pkg/types"
)

var (
	_ types.Configurator = (*LoadBalancerConfiguration)(nil)
)

type LoadBalancerConfiguration struct {
	safe.NoCopy
	Log          *log.Options          `yaml:"log,omitempty"`
	ClientSet    *clientset.Options    `yaml:"client_set,omitempty"`
	LoadBalancer *loadbalancer.Options `yaml:"load_balancer,omitempty"`
}

func (c *LoadBalancerConfiguration) Validate() error {
	if err := c.Log.Validate(); err != nil {
		return err
	}

	if err := c.ClientSet.Validate(); err != nil {
		return err
	}

	if err := c.LoadBalancer.Validate(); err != nil {
		return err
	}

	return nil
}

func (c *LoadBalancerConfiguration) SetDefaults() {
	c.Log.SetDefaults()
	c.ClientSet.SetDefaults()
	c.LoadBalancer.SetDefaults()
}

func (c *LoadBalancerConfiguration) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("", pflag.ContinueOnError)
	fs.AddFlagSet(c.Log.Flags())
	fs.AddFlagSet(c.ClientSet.Flags())
	fs.AddFlagSet(c.LoadBalancer.Flags())
	return fs
}

// NewLoadBalancerConfiguration create NewLoadBalancerConfiguration
func NewLoadBalancerConfiguration() *LoadBalancerConfiguration {
	return &LoadBalancerConfiguration{
		Log:          log.NewOptions(),
		ClientSet:    clientset.NewOptions(),
		LoadBalancer: loadbalancer.NewOptions(),
	}
}
