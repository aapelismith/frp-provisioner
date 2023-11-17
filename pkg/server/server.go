package server

import (
	"context"
	"fmt"
	"github.com/frp-sigs/frp-provisioner/pkg/api/v1beta1"
	"github.com/frp-sigs/frp-provisioner/pkg/config"
	"github.com/frp-sigs/frp-provisioner/pkg/controller"
	"github.com/frp-sigs/frp-provisioner/pkg/log"
	webhook2 "github.com/frp-sigs/frp-provisioner/pkg/webhook"
	"go.uber.org/zap"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"net"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"strconv"
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
	mgr ctrl.Manager
	cfg *config.Configuration
}

// Start the frp-provisioner controller server
func (s *Server) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("Starting frp-provisioner controller")

	if err := s.mgr.Start(ctx); err != nil {
		logger.With(zap.Error(err)).Error("Unable running frp-provisioner controller")
		return fmt.Errorf("unable running frp-provisioner controller, got: %w", err)
	}
	return nil
}

// New create frp-provisioner controller server
func New(ctx context.Context, cfg *config.Configuration) (*Server, error) {
	logger := log.FromContext(ctx)

	webhookHost, port, err := net.SplitHostPort(cfg.Manager.WebhookBindAddress)
	if err != nil {
		logger.With(zap.Error(err)).Error("unable to split host and port")
		return nil, fmt.Errorf("unable to split host and port, got: '%w'", err)
	}

	webhookPort := webhook.DefaultPort
	if port == "" {
		webhookPort, err = strconv.Atoi(port)
		if err != nil {
			logger.With(zap.Error(err)).Error("unable to convert port to number")
			return nil, fmt.Errorf("unable to convert port to number, got: '%w'", err)
		}
	}

	webhookOpts := webhook.Options{
		Host:         webhookHost,
		Port:         webhookPort,
		CertDir:      cfg.Manager.WebhookCertDir,
		CertName:     cfg.Manager.WebhookCertName,
		KeyName:      cfg.Manager.WebhookKeyName,
		ClientCAName: cfg.Manager.WebhookClientCAName,
	}

	metricsOpts := metricsserver.Options{
		CertDir:       cfg.Manager.MetricsCertDir,
		CertName:      cfg.Manager.MetricsCertName,
		KeyName:       cfg.Manager.MetricsKeyName,
		SecureServing: cfg.Manager.MetricsSecureServing,
		BindAddress:   cfg.Manager.MetricsBindAddress,
	}

	opts := ctrl.Options{
		Scheme:                        scheme,
		LeaderElection:                cfg.Manager.LeaderElection,
		LeaderElectionResourceLock:    cfg.Manager.LeaderElectionResourceLock,
		LeaderElectionNamespace:       cfg.Manager.LeaderElectionNamespace,
		LeaderElectionID:              cfg.Manager.LeaderElectionID,
		LeaderElectionReleaseOnCancel: cfg.Manager.LeaderElectionReleaseOnCancel,
		LeaseDuration:                 &cfg.Manager.LeaseDuration,
		RenewDeadline:                 &cfg.Manager.RenewDeadline,
		RetryPeriod:                   &cfg.Manager.RetryPeriod,
		Metrics:                       metricsOpts,
		WebhookServer:                 webhook.NewServer(webhookOpts),
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

	mgr, err := ctrl.NewManager(kubeConfig, opts)
	if err != nil {
		logger.With(zap.Error(err)).Error("unable to start manager")
		return nil, fmt.Errorf("unable to start manager, got: '%w'", err)
	}

	if err := (&controller.ServerReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		logger.With(zap.Error(err), zap.String("controller",
			"ServerReconciler")).Error("unable to setup server reconciler")
		return nil, fmt.Errorf("unable to setup server reconciler, got: %w", err)
	}

	if err = (&webhook2.FrpServer{}).SetupWebhookWithManager(mgr); err != nil {
		logger.With(zap.Error(err), zap.String("webhook",
			"FrpServer")).Error("unable to create webhook")
		return nil, fmt.Errorf("unable to setup FrpServer webhook, got: %w", err)
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
