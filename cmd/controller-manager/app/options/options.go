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

package options

import (
	"github.com/aapelismith/frp-provisioner/pkg/utils"
	"github.com/spf13/pflag"
	"os"
)

import (
	"errors"
)

const defaultConfigFile = "/etc/frp-provisioner/config.toml"

// ProvisionerFlags contains the command line parameters of the frp-provisioner.
// If any of the following conditions are met, the configuration field should be in CompassFlags instead of CompassConfiguration:
//   - During the lifetime of a node, its value will never or cannot be changed safely, or
//   - its value cannot be shared securely between nodes at the same time (eg: hostname);
//   - Configuration is designed to be shared between nodes.
//
// In general, please try to avoid adding tags or configuration fields,
// Because we already have a lot of confusing things.
type ProvisionerFlags struct {
	ConfigFile  string
	ShowVersion bool
}

// Validate Verify that the structure meets the requirements
func (f *ProvisionerFlags) Validate() error {
	if f.ConfigFile == "" {
		return errors.New("config file required")
	}
	_, err := os.Stat(f.ConfigFile)
	if err != nil {
		return err
	}
	return nil
}

// SetDefaults sets the default values.
func (f *ProvisionerFlags) SetDefaults() {
	f.ConfigFile = utils.EmptyOr(f.ConfigFile, defaultConfigFile)
}

// AddFlags  adds flags to the specified FlagSet
func (f *ProvisionerFlags) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&f.ShowVersion, "version", f.ShowVersion, "Print version information and exit.")
	fs.StringVar(&f.ConfigFile, "config", f.ConfigFile, "The Server will load its initial configuration from this file.")
}

// NewProvisionerFlags A new NewProvisionerFlags structure will be created
func NewProvisionerFlags() *ProvisionerFlags {
	return &ProvisionerFlags{}
}
