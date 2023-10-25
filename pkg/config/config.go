/*
 * Copyright 2021 Aapeli.Smith<aapeli.nian@gmail.com>.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package config

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/aapelismith/frp-provisioner/pkg/log"
	"github.com/aapelismith/frp-provisioner/pkg/utils"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/json"
)

var _ pflag.Value = (*serverOptionsSlice)(nil)

func isSliceJSON(data []byte) bool {
	x := bytes.TrimLeft(data, " \t\r\n")
	return len(x) > 0 && x[0] == '['
}

type serverOptionsSlice struct {
	changed bool
	value   *[]ClientCommonConfig
}

func (s *serverOptionsSlice) String() string {
	if !s.changed {
		return "[]"
	}
	data, _ := json.Marshal(s)
	return string(data)
}

func (s *serverOptionsSlice) Set(val string) error {
	opts := make([]ClientCommonConfig, 0)
	if isSliceJSON([]byte(val)) {
		opts := make([]ClientCommonConfig, 0)
		err := json.Unmarshal([]byte(val), &opts)
		if err != nil {
			return err
		}
	} else {
		opt := ClientCommonConfig{}
		err := json.Unmarshal([]byte(val), &opt)
		if err != nil {
			return err
		}
		opts = append(opts, opt)
	}
	if s.changed {
		*s.value = append(*s.value, opts...)
	} else {
		*s.value = opts
	}
	s.changed = true
	return nil
}

func (s *serverOptionsSlice) Type() string {
	return "serverOptionsSlice"
}

func newServerOptionsSlice(val *[]ClientCommonConfig) *serverOptionsSlice {
	return &serverOptionsSlice{value: val}
}

// FrpOptions is the options for frp service.
type FrpOptions struct {
	// SchedulePolicy is the policy for frpc- to schedule frpc.
	DefaultGroup string `yaml:"defaultGroup" json:"defaultGroup"`
	// Servers is the config list for frpc common config.
	Servers []ClientCommonConfig `yaml:"servers" json:"servers"`
}

// Validate validates the frpc service options.
func (o *FrpOptions) Validate() (err error) {
	if o.DefaultGroup == "" {
		err = errors.Join(err, fmt.Errorf("defaultGroup is required field"))
	}
	if len(o.Servers) == 0 {
		err = errors.Join(err, fmt.Errorf("servers is required field"))
	}
	for _, server := range o.Servers {
		if err := server.Validate(); err != nil {
			err = errors.Join(err, fmt.Errorf("incorrect server options of "+
				"'%s:%d', got: '%w'", server.ServerAddr, server.ServerPort, err))
		}
	}
	return err
}

// SetDefaults set default values for frp service options.
func (o *FrpOptions) SetDefaults() {
	if o.Servers == nil {
		o.Servers = make([]ClientCommonConfig, 0)
	}
	for index := range o.Servers {
		o.Servers[index].SetDefaults()
	}
	o.DefaultGroup = utils.EmptyOr(o.DefaultGroup, "default")
}

// AddFlags add related command line parameters
func (o *FrpOptions) AddFlags(fs *pflag.FlagSet) {
	fs.Var(newServerOptionsSlice(&o.Servers), "frp.servers", "Frp server list, array in json format, example "+
		`'[{"serverAddr":"0.0.0.0","serverPort":7000,"auth":{"token":"test-token"},"transport":{"tls":{"enable":true}}]'`)

	fs.StringVar(&o.DefaultGroup, "frp.default-group", o.DefaultGroup, "Set the default frp servers group,"+
		" and if a particular service needs a different FRP server group, simply modify the 'service.beta.kubernetes.io"+
		"/frp-group' value in its annotations. This will specify which server group the current service should use.")
}

// NewFrpOptions create and return FrpOptions
func NewFrpOptions() *FrpOptions {
	return &FrpOptions{
		DefaultGroup: "",
		Servers:      make([]ClientCommonConfig, 0),
	}
}

// Configuration is the controller configuration.
type Configuration struct {
	// Log is the log options struct for zap logger
	Log *log.Options `yaml:"log,omitempty" json:"log,omitempty"`
	// Frp is the frp options for current server
	Frp *FrpOptions `yaml:"frp,omitempty" json:"frp,omitempty"`
	// Manager is the controller-manager options for controller-runtime
	Manager *ManagerOptions `yaml:"manager,omitempty" json:"manager,omitempty"`
}

// AddFlags adds flags for a specific configuration to the specified FlagSet
func (c *Configuration) AddFlags(fs *pflag.FlagSet) {
	c.Log.AddFlags(fs)
	c.Frp.AddFlags(fs)
	c.Manager.AddFlags(fs)
}

// SetDefaults sets the default values for a specific configuration.
func (c *Configuration) SetDefaults() {
	c.Log.SetDefaults()
	c.Frp.SetDefaults()
	c.Manager.SetDefaults()
}

// Validate validates a specific configuration.
func (c *Configuration) Validate() (err error) {
	if err := c.Log.Validate(); err != nil {
		err = errors.Join(err, fmt.Errorf("invalid log config, got: '%w'", err))
	}
	if err := c.Frp.Validate(); err != nil {
		err = errors.Join(err, fmt.Errorf("invalid frp config, got: '%w'", err))
	}
	if err := c.Manager.Validate(); err != nil {
		err = errors.Join(err, fmt.Errorf("invalid manager config, got: '%w'", err))
	}
	return err
}

// NewConfiguration create Configuration
func NewConfiguration() *Configuration {
	return &Configuration{
		Log:     log.NewOptions(),
		Frp:     NewFrpOptions(),
		Manager: NewManagerOptions(),
	}
}
