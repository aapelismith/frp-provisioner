/*
 * Copyright 2021 The Frp Sig Authors.
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
	"errors"
	"fmt"
	"github.com/frp-sigs/frp-provisioner/pkg/log"
	"github.com/spf13/pflag"
)

// Configuration is the controller configuration.
type Configuration struct {
	// Log is the log options struct for zap logger
	Log *log.Options `yaml:"log,omitempty" json:"log,omitempty"`
	// Manager is the controller-manager options for controller-runtime
	Manager *ManagerOptions `yaml:"manager,omitempty" json:"manager,omitempty"`
}

// AddFlags adds flags for a specific configuration to the specified FlagSet
func (c *Configuration) AddFlags(fs *pflag.FlagSet) {
	c.Log.AddFlags(fs)
	c.Manager.AddFlags(fs)
}

// SetDefaults sets the default values for a specific configuration.
func (c *Configuration) SetDefaults() {
	c.Log.SetDefaults()
	c.Manager.SetDefaults()
}

// Validate validates a specific configuration.
func (c *Configuration) Validate() (err error) {
	if err := c.Log.Validate(); err != nil {
		err = errors.Join(err, fmt.Errorf("invalid log config, got: '%w'", err))
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
		Manager: NewManagerOptions(),
	}
}
