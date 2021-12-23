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

package app

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/client-go/informers"
	"kunstack.com/pharos/cmd/loadbalancer/app/options"
	"kunstack.com/pharos/pkg/client/clientset"
	"kunstack.com/pharos/pkg/config"
	"kunstack.com/pharos/pkg/controller/loadbalancer"
	"kunstack.com/pharos/pkg/log"
	"kunstack.com/pharos/pkg/safe"
	"os"
	"sync"
	"time"
)

const (
	// component component name
	component     = "loadbalancer"
	shortDescribe = "Load balancing controller implemented for k8s bare metal cluster"
)

// NewLoadBalancerCommand create loadBalancer command
func NewLoadBalancerCommand(stopChan <-chan struct{}) *cobra.Command {
	cleanFlagSet := pflag.NewFlagSet(component, pflag.ContinueOnError)
	loadBalancerFlags := options.NewLoadBalancerFlags()
	loadBalancerFlags.SetDefaults()
	cfg := config.NewLoadBalancerConfiguration()
	cfg.SetDefaults()

	cmd := &cobra.Command{
		Use:                component,
		Short:              shortDescribe,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			wg := sync.WaitGroup{}

			ctx = log.WithContext(ctx)
			if err := cleanFlagSet.Parse(args); err != nil {
				return err
			}

			// check if there are non-flag arguments in the command line
			restArgs := cleanFlagSet.Args()
			if len(restArgs) > 0 {
				return fmt.Errorf("unknown command: %s", restArgs[0])
			}

			help, err := cleanFlagSet.GetBool("help")
			if err != nil {
				return fmt.Errorf(`"help" flag is non-bool, programmer error, please correct`)
			}
			if help {
				return cmd.Help()
			}

			if err := loadBalancerFlags.Validate(); err != nil {
				return err
			}

			err = options.LoadConfigFile(loadBalancerFlags.LoadBalancerConfig, cfg)
			if err != nil {
				return err
			}

			if err := options.LoadBalancerFlagPrecedence(args, cfg); err != nil {
				return err
			}

			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("config file is incorrect: %v", err)
			}

			lvl, err := log.ParseLevel(cfg.Log.Level)
			if err != nil {
				return err
			}
			log.SetLevel(lvl)

			encoder, err := log.ParseEncoder(cfg.Log.Format)
			if err != nil {
				return err
			}
			log.SetEncoder(encoder)

			cal, err := log.ParseCaller(cfg.Log.Caller)
			if err != nil {
				return err
			}
			log.SetCallerEncoder(cal)

			timeEncoder, err := log.ParseTimeEncoder(cfg.Log.Time)
			if err != nil {
				return err
			}
			log.SetTimeEncoder(timeEncoder)

			if cfg.Log.File != "" {
				file, err := os.OpenFile(cfg.Log.File, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
				if err != nil {
					return err
				}
				log.SetOutput(file)
			}

			cli, err := clientset.NewClient(cfg.ClientSet)
			if err != nil {
				log.Errorln(err)
				return err
			}

			informer := informers.NewSharedInformerFactory(cli, time.Minute*60)

			ctl, err := loadbalancer.NewController(ctx, cfg.LoadBalancer, informer.Core().V1().Services(), informer.Apps().V1().DaemonSets(), cli)
			if err != nil {
				log.Errorln(err)
				return err
			}

			wg.Add(2)

			safe.Go(func() {
				defer wg.Done()
				ctl.Run(stopChan, 3)
			})

			safe.Go(func() {
				defer wg.Done()
				informer.Start(stopChan)
			})

			wg.Wait()
			return nil
		},
	}
	cleanFlagSet.BoolP("help", "h", false, fmt.Sprintf("Display help information for command %s", cmd.Name()))
	cleanFlagSet.AddFlagSet(loadBalancerFlags.Flags())
	cleanFlagSet.AddFlagSet(cfg.Flags())
	cmd.Flags().AddFlagSet(cleanFlagSet) // In order to --help can display content
	return cmd
}
