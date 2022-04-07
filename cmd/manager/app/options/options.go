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
	"fmt"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"kunstack.com/pharos/pkg/client/clientset"
	"kunstack.com/pharos/pkg/client/helm"
	"kunstack.com/pharos/pkg/controller/ingress"
	"kunstack.com/pharos/pkg/log"
	"os"
	"strings"
	"time"
)

var allControllerNameSelectors []string

const defaultConfigFile = "/etc/manager/config.yaml"

// ControllerManagerOptions contains the command line parameters of the ControllerManager.
type ControllerManagerOptions struct {
	ConfigFile string
	// ControllerGates is the list of controller gates to enable or disable controller.
	// '*' means "all enabled by default controllers"
	// 'foo' means "enable 'foo'"
	// '-foo' means "disable 'foo'"
	// first item for a particular name wins.
	//     e.g. '-foo,foo' means "disable foo", 'foo,-foo' means "enable foo"
	// * has the lowest priority.
	//     e.g. *,-foo, means "disable 'foo'"
	ControllerGates          []string           `yaml:"controllerGates,omitempty"`
	Log                      *log.Options       `yaml:"log,omitempty"`
	Helm                     *helm.Options      `yaml:"helm,omitempty"`
	ClientSet                *clientset.Options `yaml:"clientSet,omitempty"`
	Ingress                  *ingress.Options   `yaml:"ingress,omitempty"`
	LeaderElect              bool               `yaml:"leaderElect,omitempty"`
	LeaderElectLeaseDuration time.Duration      `yaml:"leaderElectLeaseDuration,omitempty"`
	LeaderElectRenewDeadline time.Duration      `yaml:"leaderElectRenewDeadline,omitempty"`
	LeaderElectRetryPeriod   time.Duration      `yaml:"leaderElectRetryPeriod,omitempty"`
}

// Validate Verify that the structure meets the requirements
func (o *ControllerManagerOptions) Validate() error {
	if o.ConfigFile == "" {
		return errors.New("config file required")
	}
	info, err := os.Stat(o.ConfigFile)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("config should not be a regular file")
	}
	if err := o.ClientSet.Validate(); err != nil {
		return fmt.Errorf("clientset: %v", err)
	}
	return nil
}

// Complete .
func (o *ControllerManagerOptions) Complete() error {
	override, err := yaml.Marshal(o)
	if err != nil {
		fmt.Println(err, 1)
		return err
	}

	data, err := ioutil.ReadFile(o.ConfigFile)
	if err != nil {
		fmt.Println(err, 2)
		return err
	}

	if err = yaml.UnmarshalStrict(data, o); err != nil {
		fmt.Println(err, 3)
		return err
	}

	return yaml.UnmarshalStrict(override, o)
}

// NewControllerManagerOptions A new ControllerManagerOptions structure will be created
// and filled with default values
func NewControllerManagerOptions() *ControllerManagerOptions {
	return &ControllerManagerOptions{
		ConfigFile:               defaultConfigFile,
		Log:                      log.NewOptions(),
		ControllerGates:          []string{"*"},
		Helm:                     helm.NewOptions(),
		Ingress:                  ingress.NewOptions(),
		ClientSet:                clientset.NewOptions(),
		LeaderElect:              false,
		LeaderElectLeaseDuration: 30 * time.Second,
		LeaderElectRenewDeadline: 15 * time.Second,
		LeaderElectRetryPeriod:   5 * time.Second,
	}
}

// Flags Get the flags of ControllerManagerOptions
func (o *ControllerManagerOptions) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("", pflag.ContinueOnError)
	fs.AddFlagSet(o.Log.Flags())
	fs.AddFlagSet(o.Helm.Flags())
	fs.AddFlagSet(o.ClientSet.Flags())
	fs.AddFlagSet(o.Ingress.Flags())

	fs.StringVar(&o.ConfigFile, "config", o.ConfigFile, "The Server will load its initial configuration from this file.")

	fs.StringSliceVar(&o.ControllerGates, "controllers", o.ControllerGates, fmt.Sprintf(""+
		"A list of controllers to enable. '*' enables all on-by-default controllers, 'foo' enables the controller "+
		"named 'foo', '-foo' disables the controller named 'foo'.\nAll controllers: %s",
		strings.Join(allControllerNameSelectors, ", ")))

	fs.DurationVar(&o.LeaderElectLeaseDuration, "leader-election.lease-duration",
		o.LeaderElectLeaseDuration, "The duration that non-leader candidates will wait "+
			"after observing a leadership renewal until attempting to acquire leadership of a led "+
			"but unRenewed leader slot. This is effectively the maximum duration that a leader can be"+
			" stopped before it is replaced by another candidate. This is only applicable if leader "+
			"election is enabled.")

	fs.DurationVar(&o.LeaderElectRenewDeadline, "leader-election.renew-deadline",
		o.LeaderElectRenewDeadline, "The interval between attempts by the acting "+
			"master to renew a leadership slot before it stops leading. This must be less "+
			"than or equal to the lease duration. This is only applicable if leader election is enabled.")

	fs.DurationVar(&o.LeaderElectRetryPeriod, "leader-election.retry-period",
		o.LeaderElectRetryPeriod, "The duration the clients should wait between "+
			"attempting acquisition and renewal of a leadership. This is only applicable if leader election is enabled.")
	return fs
}
