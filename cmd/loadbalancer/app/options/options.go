/*
 * Copyright 2021 The KunStack Authors.
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

package options

import (
	"errors"
	"os"

	"github.com/spf13/pflag"
)

const defaultConfigFile = "/etc/loadbalancer.yaml"

// LoadBalancerFlags contains the command line parameters of the LoadBalancer.
// If any of the following conditions are met, the configuration field should be in LoadBalancerFlags instead of LoadBalancerConfiguration:
//	- During the lifetime of a node, its value will never or cannot be changed safely, or
//	- its value cannot be shared securely between nodes at the same time (eg: hostname);
// 	- Configuration is designed to be shared between nodes.
// In general, please try to avoid adding tags or configuration fields,
// Because we already have a lot of confusing things.
type LoadBalancerFlags struct {
	LoadBalancerConfig string
}

// Validate Verify that the structure meets the requirements
func (f *LoadBalancerFlags) Validate() error {
	if f.LoadBalancerConfig == "" {
		return errors.New("config file required")
	}
	info, err := os.Stat(f.LoadBalancerConfig)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return errors.New("config should not be a directory")
	}
	return nil
}

// SetDefaults sets the default values.
func (f *LoadBalancerFlags) SetDefaults() {
	f.LoadBalancerConfig = defaultConfigFile
}

// NewLoadBalancerFlags A new LoadBalancerFlags structure will be created
// and filled with default values
func NewLoadBalancerFlags() *LoadBalancerFlags {
	return &LoadBalancerFlags{}
}

// Flags Get the flags of LoadBalancerFlags
func (f *LoadBalancerFlags) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("", pflag.ExitOnError)
	fs.StringVar(&f.LoadBalancerConfig, "config", f.LoadBalancerConfig, "The Server will load its initial configuration from this file.")
	return fs
}
