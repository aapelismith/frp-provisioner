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

package options

import (
	"github.com/frp-sigs/frp-provisioner/pkg/config"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/yaml"
	"os"
)

// FlagPrecedence The Flag of the frp-provisioner is parsed again.
// The goal is for the data specified in flag to override the data in the configuration file
func FlagPrecedence(args []string, c *config.Configuration) error {
	cleanFlags := pflag.NewFlagSet("", pflag.ContinueOnError)
	NewManagerFlags().AddFlags(cleanFlags)
	c.AddFlags(cleanFlags)
	if err := cleanFlags.Parse(args); err != nil {
		return err
	}
	return nil
}

// LoadConfigFile Load the configuration file from disk and populate the structure Configuration
func LoadConfigFile(filename string, c *config.Configuration) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return yaml.UnmarshalStrict(data, c)
}
