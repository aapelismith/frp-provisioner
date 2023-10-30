package server

import (
	"context"
	"fmt"
	"github.com/frp-sigs/frp-provisioner/pkg/api/v1beta1"
	"github.com/frp-sigs/frp-provisioner/pkg/config"
	"github.com/frp-sigs/frp-provisioner/pkg/controller/service"
	"github.com/frp-sigs/frp-provisioner/pkg/log"
	"go.uber.org/zap"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// default re-sync period for all informer factories
const defaultRsync = 600 * time.Second

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(v1beta1.AddToScheme(scheme))
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
}

// Server frp controller server
type Server struct {
	mgr      ctrl.Manager
	cfg      *config.Configuration
	informer informers.SharedInformerFactory
}

// Start the frp-provisioner controller server
func (s *Server) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("Starting frp-provisioner controller")

	s.informer.Start(ctx.Done())
	defer s.informer.Shutdown()

	if err := s.mgr.Start(ctx); err != nil {
		logger.With(zap.Error(err)).Error("Unable running frp-provisioner controller")
		return fmt.Errorf("unable running frp-provisioner controller, got: %w", err)
	}
	return nil
}

// New create frp-provisioner controller server
func New(ctx context.Context, cfg *config.Configuration) (*Server, error) {
	logger := log.FromContext(ctx)

	metricsOpts := metricsserver.Options{
		CertDir:       cfg.Manager.MetricsCertDir,
		CertName:      cfg.Manager.MetricsCertName,
		KeyName:       cfg.Manager.MetricsKeyName,
		SecureServing: cfg.Manager.MetricsSecureServing,
		BindAddress:   cfg.Manager.MetricsBindAddress,
	}

	opts := ctrl.Options{
		Scheme:                        scheme,
		Metrics:                       metricsOpts,
		LeaderElection:                cfg.Manager.LeaderElection,
		LeaderElectionResourceLock:    cfg.Manager.LeaderElectionResourceLock,
		LeaderElectionNamespace:       cfg.Manager.LeaderElectionNamespace,
		LeaderElectionID:              cfg.Manager.LeaderElectionID,
		LeaderElectionReleaseOnCancel: cfg.Manager.LeaderElectionReleaseOnCancel,
		LeaseDuration:                 &cfg.Manager.LeaseDuration,
		RenewDeadline:                 &cfg.Manager.RenewDeadline,
		RetryPeriod:                   &cfg.Manager.RetryPeriod,
		HealthProbeBindAddress:        cfg.Manager.HealthProbeBindAddress,
		ReadinessEndpointName:         cfg.Manager.ReadinessEndpointName,
		LivenessEndpointName:          cfg.Manager.LivenessEndpointName,
		PprofBindAddress:              cfg.Manager.PprofBindAddress,
		GracefulShutdownTimeout:       &cfg.Manager.GracefulShutdownTimeout,
	}

	kubeConfig, err := ctrl.GetConfig()
	if err != nil {
		logger.With(zap.Error(err)).Error("unable to get kubernetes config")
		return nil, fmt.Errorf("unable to get kubernetes config, got: '%w'", err)
	}

	client, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		logger.With(zap.Error(err)).Error("Unable to create kubernetes client")
		return nil, fmt.Errorf("unable to create kubernetes client, got: '%w'", err)
	}

	informer := informers.NewSharedInformerFactory(client, defaultRsync)

	mgr, err := ctrl.NewManager(kubeConfig, opts)
	if err != nil {
		logger.With(zap.Error(err)).Error("unable to start manager")
		return nil, fmt.Errorf("unable to start manager, got: '%w'", err)
	}

	ctr, err := service.NewController(ctx, cfg.Frp, client,
		informer.Core().V1().Services(), informer.Core().V1().Nodes())
	if err != nil {
		logger.With(zap.Error(err), zap.String("controller",
			"FrpController")).Error("unable to create controller")
		return nil, fmt.Errorf("unable to create controller, got: %w", err)
	}

	if err := ctr.SetupWithManager(mgr); err != nil {
		logger.With(zap.Error(err), zap.String("controller",
			"FrpController")).Error("unable to setup controller")
		return nil, fmt.Errorf("unable to setup controller, got: %w", err)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		logger.With(zap.Error(err)).Error("unable to set up health check")
		return nil, fmt.Errorf("unable to set up health check, got: %w", err)
	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		logger.With(zap.Error(err)).Error("unable to set up ready check")
		return nil, fmt.Errorf("unable to set up ready check, got: %w", err)
	}

	return &Server{mgr: mgr, informer: informer, cfg: cfg}, nil
}
