/*
 * Copyright 2021 Aapeli <aapeli.nian@gmail.com>.
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
	"flag"
	"fmt"
	"github.com/frp-sigs/frp-provisioner/cmd/manager/app/options"
	"github.com/frp-sigs/frp-provisioner/pkg/config"
	"github.com/frp-sigs/frp-provisioner/pkg/log"
	"github.com/frp-sigs/frp-provisioner/pkg/server"
	"github.com/frp-sigs/frp-provisioner/pkg/version"
	"github.com/go-logr/zapr"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	// component component name
	component     = "frp-provisioner-manager"
	shortDescribe = "A component manager for frp-provisioner to help you expose kubernetes service behind a NAT or firewall to the internet."
)

// NewManagerCommand create a new *cobra.Command for frp-provisioner-manager
func NewManagerCommand(baseCtx context.Context) *cobra.Command {
	cleanFlagSet := pflag.NewFlagSet(component, pflag.ContinueOnError)
	managerFlags := options.NewManagerFlags()
	managerFlags.SetDefaults()
	cfg := config.NewConfiguration()
	cfg.SetDefaults()

	cmd := &cobra.Command{
		Use:                component,
		Short:              shortDescribe,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithCancel(baseCtx)
			defer cancel()
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
			if managerFlags.ShowVersion {
				cmd.Println(version.Get())
				return nil
			}
			if err := managerFlags.Validate(); err != nil {
				return err
			}
			if err := options.LoadConfigFile(managerFlags.ConfigFile, cfg); err != nil {
				return fmt.Errorf("config file %s contains errors: %v", managerFlags.ConfigFile, err)
			}
			if err := options.FlagPrecedence(args, cfg); err != nil {
				return err
			}
			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("config file is incorrect: %v", err)
			}
			logger, err := log.NewLogger(ctx, cfg.Log)
			if err != nil {
				return fmt.Errorf("cannot create logger: %v", err)
			}
			log.ReplaceGlobals(logger)
			ctrl.SetLogger(zapr.NewLogger(logger))
			ctx = log.NewContext(ctx, logger)
			srv, err := server.NewManagerServer(ctx, cfg)
			if err != nil {
				return fmt.Errorf("cannot create frp-provisioner server: %v", err)
			}
			return srv.Start(ctx)
		},
	}

	cleanFlagSet.BoolP("help", "h", false, fmt.Sprintf("Display help information for command %s", cmd.Name()))
	managerFlags.AddFlags(cleanFlagSet)
	cfg.AddFlags(cleanFlagSet)
	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	cmd.Flags().AddFlagSet(cleanFlagSet) // In order to --help can display content
	return cmd
}
