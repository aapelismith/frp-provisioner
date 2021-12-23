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
	"io/ioutil"
	"os"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
	"kunstack.com/pharos/pkg/config"
)

// LoadBalancerFlagPrecedence The Flag of the LoadBalancer is parsed again.
// The goal is for the data specified in flag to override the data in the configuration file
func LoadBalancerFlagPrecedence(args []string, c *config.LoadBalancerConfiguration) error {
	cleanFlags := pflag.NewFlagSet("", pflag.ContinueOnError)
	cleanFlags.AddFlagSet(NewLoadBalancerFlags().Flags())
	cleanFlags.AddFlagSet(c.Flags())
	if err := cleanFlags.Parse(args); err != nil {
		return err
	}
	return nil
}

// LoadConfigFile Load the configuration file from disk and populate the structure Configuration
func LoadConfigFile(filename string, c *config.LoadBalancerConfiguration) error {
	file, err := os.OpenFile(filename, os.O_RDONLY, 0600)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()
	stream, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}
	err = yaml.UnmarshalStrict(stream, c)
	if err != nil {
		return err
	}
	return nil
}
