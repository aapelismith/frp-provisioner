package server

import (
	"context"
	"fmt"
	"github.com/aapelismith/frp-provisioner/pkg/config"
	"github.com/aapelismith/frp-provisioner/pkg/controller"
	"github.com/aapelismith/frp-provisioner/pkg/log"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
}

// Server frp controller server
type Server struct {
	mgr ctrl.Manager
	cfg *config.Configuration
}

// Start the frp controller server
func (s *Server) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("starting frp controller server")

	if err := s.mgr.Start(ctx); err != nil {
		logger.With(zap.Error(err)).Error("problem running controller server")
		return fmt.Errorf("problem running controller server, got: %w", err)
	}
	return nil
}

// New create frp controller server
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

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), opts)
	if err != nil {
		logger.With(zap.Error(err)).Error("unable to start manager")
		return nil, fmt.Errorf("unable to start manager, got: '%w'", err)
	}

	ctr, err := controller.NewController(cfg.Frp)
	if err != nil {
		logger.With(zap.Error(err), zap.String("controller",
			"ServiceController")).Error("unable to create controller")
		return nil, fmt.Errorf("unable to create controller, got: %w", err)
	}

	if err := ctr.SetupWithManager(mgr); err != nil {
		logger.With(zap.Error(err), zap.String("controller",
			"ServiceController")).Error("unable to setup controller")
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

	return &Server{mgr: mgr, cfg: cfg}, nil
}
