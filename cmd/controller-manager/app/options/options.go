package options

import (
	"github.com/aapelismith/frp-service-provider/pkg/log"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/errors"
)

// Configuration is the controller-manager configuration.
type Configuration struct {
	Log *log.Options `yaml:"log,omitempty" json:"log,omitempty"`
}

// AddFlags adds flags for a specific configuration to the specified FlagSet
func (c *Configuration) AddFlags(fs *pflag.FlagSet) {
	c.Log.AddFlags(fs)
}

// SetDefaults sets the default values for a specific configuration.
func (c *Configuration) SetDefaults() {
	c.Log.SetDefaults()
}

// Validate validates a specific configuration.
func (c *Configuration) Validate() error {
	errs := make([]error, 0)

	if err := c.Log.Validate(); err != nil {
		errs = append(errs, err)
	}

	return errors.NewAggregate(errs)
}
