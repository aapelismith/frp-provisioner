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

package app

import (
	"context"
	"flag"
	"fmt"
	"github.com/aapelismith/frp-provisioner/cmd/controller-manager/app/options"
	"github.com/aapelismith/frp-provisioner/pkg/config"
	"github.com/aapelismith/frp-provisioner/pkg/log"
	"github.com/aapelismith/frp-provisioner/pkg/server"
	"github.com/go-logr/zapr"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	// component component name
	component     = "frp-provisioner"
	shortDescribe = "A fast reverse proxy to help you expose kubernetes service behind a NAT or firewall to the internet."
)

// NewProvisionerCommand create a new *cobra.Command provisioner
func NewProvisionerCommand(baseCtx context.Context) *cobra.Command {
	cleanFlagSet := pflag.NewFlagSet(component, pflag.ContinueOnError)
	provisionerFlags := options.NewProvisionerFlags()
	provisionerFlags.SetDefaults()
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
			if err := provisionerFlags.Validate(); err != nil {
				return err
			}
			strictErrors, err := options.LoadConfigFile(provisionerFlags.ConfigFile, cfg)
			if err != nil {
				return fmt.Errorf("config file %s contains errors: %v", provisionerFlags.ConfigFile, err)
			}
			if len(strictErrors) > 0 {
				return fmt.Errorf("config file %s contains strict errors: %v", provisionerFlags.ConfigFile, strictErrors)
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

			srv, err := server.New(ctx, cfg)
			if err != nil {
				return fmt.Errorf("cannot create server: %v", err)
			}
			return srv.Start(ctx)
		},
	}
	cleanFlagSet.BoolP("help", "h", false, fmt.Sprintf("Display help information for command %s", cmd.Name()))
	provisionerFlags.AddFlags(cleanFlagSet)
	cfg.AddFlags(cleanFlagSet)
	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	cmd.Flags().AddFlagSet(cleanFlagSet) // In order to --help can display content
	return cmd
}
