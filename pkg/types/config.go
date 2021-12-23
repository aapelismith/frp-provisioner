package types

import (
	"github.com/spf13/pflag"
)

// Configurator All configurators should implement this interface
type Configurator interface {
	//SetDefaults Set the default value of the configuration
	SetDefaults()
	//Validate verify that the configuration is correct
	Validate() error
	//Flags Get configured command line parameters
	Flags() *pflag.FlagSet
}
