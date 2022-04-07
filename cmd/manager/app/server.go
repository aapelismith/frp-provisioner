package app

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	clientScheme "k8s.io/client-go/kubernetes/scheme"
	"kunstack.com/pharos/cmd/manager/app/options"
	"kunstack.com/pharos/pkg/client/clientset"
	"kunstack.com/pharos/pkg/log"
	ctrl "sigs.k8s.io/controller-runtime"
	"sync"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilRuntime.Must(clientScheme.AddToScheme(scheme))
}

// NewControllerCommand create controllers command
func NewControllerCommand(baseCtx context.Context) *cobra.Command {
	cleanFlagSet := pflag.NewFlagSet("run", pflag.ContinueOnError)
	opts := options.NewControllerManagerOptions()

	cmd := &cobra.Command{
		Use:                "run",
		Short:              "Start and run controller manager",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithCancel(baseCtx)
			defer cancel()
			wg := sync.WaitGroup{}

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

			if err := opts.Complete(); err != nil {
				return err
			}

			if err := opts.Validate(); err != nil {
				return err
			}

			logger, err := log.NewFromOptions(opts.Log)
			if err != nil {
				return err
			}

			log.SetLogger(logger)
			ctrl.SetLogger(logr.New(logger.Sink()))

			logger = logger.WithFields("stage", "setup")

			conf, err := clientset.NewConfig(opts.ClientSet)
			if err != nil {
				logger.Errorf("unable create client set, got: %v", err)
				return err
			}

			mgr, err := ctrl.NewManager(conf, ctrl.Options{
				Scheme:                  scheme,
				Port:                    8443,
				LeaderElection:          opts.LeaderElect,
				LeaderElectionNamespace: "kube-system",
				LeaderElectionID:        "pharos-controller-manager-leader-election",
				LeaseDuration:           &opts.LeaderElectLeaseDuration,
				RetryPeriod:             &opts.LeaderElectRetryPeriod,
				RenewDeadline:           &opts.LeaderElectRenewDeadline,
			})
			if err != nil {
				logger.Errorf("unable create controller manager, got: %v", err)
				return err
			}

			if err := mgr.Start(ctx); err != nil {
				cancel() // cancel the context
				logger.Errorf("problem running manager,got: %v", err)
				return err
			}
			wg.Wait()
			return nil
		},
	}
	cleanFlagSet.BoolP("help", "h", false, fmt.Sprintf("Display help information for command %s", cmd.Name()))
	cleanFlagSet.AddFlagSet(opts.Flags())
	cmd.Flags().AddFlagSet(cleanFlagSet) // In order to --help can display content
	return cmd
}
