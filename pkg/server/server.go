package server

import (
	"context"
	"fmt"
	"github.com/frp-sigs/frp-provisioner/pkg/api/v1beta1"
	"github.com/frp-sigs/frp-provisioner/pkg/config"
	"github.com/frp-sigs/frp-provisioner/pkg/controller"
	"github.com/frp-sigs/frp-provisioner/pkg/utils/fieldindex"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"net"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"strconv"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(v1beta1.AddToScheme(scheme))
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
}

// ManagerServer frp controller server
type ManagerServer struct {
	mgr ctrl.Manager
	cfg *config.Configuration
}

// Start the frp-provisioner controller server
func (s *ManagerServer) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("Starting frp-provisioner controller")

	if err := s.mgr.Start(ctx); err != nil {
		logger.Error(err, "Unable running frp-provisioner controller")
		return fmt.Errorf("unable running frp-provisioner controller, got: %w", err)
	}
	return nil
}

// NewManagerServer create frp-provisioner controller server
func NewManagerServer(ctx context.Context, cfg *config.Configuration) (*ManagerServer, error) {
	logger := log.FromContext(ctx)
	webhookHost, port, err := net.SplitHostPort(cfg.Manager.WebhookBindAddress)
	if err != nil {
		logger.Error(err, "unable to split host and port")
		return nil, fmt.Errorf("unable to split host and port, got: '%w'", err)
	}
	webhookPort := webhook.DefaultPort
	if port == "" {
		webhookPort, err = strconv.Atoi(port)
		if err != nil {
			logger.Error(err, "unable to convert port to number")
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
		PprofBindAddress:              cfg.Manager.PprofBindAddress,
		GracefulShutdownTimeout:       &cfg.Manager.GracefulShutdownTimeout,
	}
	kubeConfig, err := ctrl.GetConfig()
	if err != nil {
		logger.Error(err, "unable to get kubernetes config")
		return nil, fmt.Errorf("unable to get kubernetes config, got: '%w'", err)
	}
	mgr, err := ctrl.NewManager(kubeConfig, opts)
	if err != nil {
		logger.Error(err, "unable to start manager")
		return nil, fmt.Errorf("unable to start manager, got: '%w'", err)
	}
	err = fieldindex.RegisterFieldIndexes(ctx, mgr.GetCache())
	if err != nil {
		logger.Error(err, "unable  Register Field Indexes to cache")
		return nil, fmt.Errorf("unable  RegisterFieldIndexes to cache got: '%w'", err)
	}
	if err := (&controller.ServiceReconciler{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		Options: cfg.Manager,
	}).SetupWithManager(mgr); err != nil {
		logger.Error(err, "unable to setup server reconciler", "controller", "ServiceReconciler")
		return nil, fmt.Errorf("unable to setup server reconciler, got: %w", err)
	}
	if err := (&controller.FrpServerReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		logger.Error(err, "unable to setup frpserver reconciler", "controller", "FrpServerReconciler")
		return nil, fmt.Errorf("unable to setup frpserver reconciler, got: %w", err)
	}
	if err = (&controller.FrpServerValidator{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWebhookWithManager(mgr); err != nil {
		logger.Error(err, "unable to create webhook", "webhook", "FrpServerValidator")
		return nil, fmt.Errorf("unable to setup FrpServerValidator webhook, got: %w", err)
	}
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		logger.Error(err, "unable to set up health check")
		return nil, fmt.Errorf("unable to set up health check, got: %w", err)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		logger.Error(err, "unable to set up ready check")
		return nil, fmt.Errorf("unable to set up ready check, got: %w", err)
	}
	return &ManagerServer{mgr: mgr, cfg: cfg}, nil
}
