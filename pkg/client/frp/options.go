package frp

import (
	"fmt"
	"github.com/fatedier/frp/pkg/config"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/errors"
)

const defaultWorkspace = "/tmp/frp-service-provider"

// ServerConfig is the config for frp server.
type ServerConfig struct {
	// ServerName is the name of server.
	ServerName string `json:"server_name"`
	// ClientConfig is the config for frpc.
	config.ClientCommonConf `yaml:",inline" json:",inline"`
}

// Options is the options for frp client.
type Options struct {
	// Workspace is the workspace for frpc.
	Workspace string `yaml:"workspace" json:"workspace"`
	// Servers is the config list for frpc common config.
	Servers []ServerConfig `yaml:"servers" json:"servers"`
}

// Validate validates the frp client options.
func (o *Options) Validate() error {
	errs := make([]error, 0)
	if o.Workspace == "" {
		errs = append(errs, fmt.Errorf("workspace cannot be empty"))
	}
	if len(o.Servers) == 0 {
		errs = append(errs, fmt.Errorf("servers cannot be empty"))
	}
	for index, srv := range o.Servers {
		if srv.ServerName == "" {
			errs = append(errs, fmt.Errorf("servers[%d].server_name cannot be empty", index))
		}
		if err := srv.ClientCommonConf.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("validate server[%d] failed and got: '%w'", index, err))
		}
	}
	return errors.NewAggregate(errs)
}

// SetDefaults set default values for frp client options.
func (o *Options) SetDefaults() {
	if o.Workspace == "" {
		o.Workspace = defaultWorkspace
	}
	if o.Servers == nil {
		o.Servers = make([]ServerConfig, 0)
	}
}

// AddFlags add related command line parameters
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Workspace, "frpc.workspace", o.Workspace, "The workspace for "+
		"frpc-service-provider. config file will be generated in this directory.")
}
