package config

import (
	"errors"
	"fmt"
	"github.com/frp-sigs/frp-provisioner/pkg/utils"
	"github.com/spf13/pflag"
	"time"
)

const (
	defaultLeaderElectionResourceLock = "leases"
	defaultLeaderElectionID           = "frp-provisioner"
	defaultLeaseDuration              = 15 * time.Second
	defaultRenewDeadline              = 10 * time.Second
	defaultRetryPeriod                = 2 * time.Second
	defaultGracefulShutdownPeriod     = 30 * time.Second
	defaultReadinessEndpoint          = "/readyz"
	defaultLivenessEndpoint           = "/healthz"
)

// ManagerOptions  contains the configuration for the manager.
type ManagerOptions struct {
	// LeaderElection determines whether to use leader election when
	// starting the manager.
	LeaderElection bool `json:"leaderElection" yaml:"leaderElection"`

	// LeaderElectionResourceLock determines which resource lock to use for leader election,
	// defaults to "leases". Change this value only if you know what you are doing.
	LeaderElectionResourceLock string `json:"leaderElectionResourceLock" yaml:"leaderElectionResourceLock"`

	// LeaderElectionNamespace determines the namespace in which the leader
	// election resource will be created.
	LeaderElectionNamespace string `json:"leaderElectionNamespace" yaml:"leaderElectionNamespace"`

	// LeaderElectionID determines the name of the resource that leader election
	// will use for holding the leader lock.
	LeaderElectionID string `json:"leaderElectionID" yaml:"leaderElectionID"`

	// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
	// when the Manager ends. This requires the binary to immediately end when the
	// Manager is stopped, otherwise this setting is unsafe. Setting this significantly
	// speeds up voluntary leader transitions as the new leader doesn't have to wait
	// LeaseDuration time first.
	LeaderElectionReleaseOnCancel bool `json:"leaderElectionReleaseOnCancel" yaml:"leaderElectionReleaseOnCancel"`

	// LeaseDuration is the duration that non-leader candidates will
	// wait to force acquire leadership. This is measured against time of
	// last observed ack. Default is 15 seconds.
	LeaseDuration time.Duration `json:"leaseDuration" yaml:"leaseDuration"`

	// RenewDeadline is the duration that the acting controlPlane will retry
	// refreshing leadership before giving up. Default is 10 seconds.
	RenewDeadline time.Duration `json:"renewDeadline" yaml:"renewDeadline"`

	// RetryPeriod is the duration the LeaderElector clients should wait
	// between tries of actions. Default is 2 seconds.
	RetryPeriod time.Duration `json:"retryPeriod" yaml:"retryPeriod"`

	// SecureServing enables serving metrics via https.
	// Per default metrics will be served via http.
	MetricsSecureServing bool `json:"metricsSecureServing" yaml:"metricsSecureServing"`

	// BindAddress is the bind address for the metrics server.
	// It will be defaulted to ":8080" if unspecified.
	// Set this to "0" to disable the metrics server.
	MetricsBindAddress string `json:"metricsBindAddress" yaml:"metricsBindAddress"`

	// CertDir is the directory that contains the server key and certificate. Defaults to
	// <temp-dir>/k8s-metrics-server/serving-certs.
	//
	// Note: This option is only used when TLSOpts does not set GetCertificate.
	// Note: If certificate or key doesn't exist a self-signed certificate will be used.
	MetricsCertDir string `json:"metricsCertDir" yaml:"metricsCertDir"`

	// CertName is the server certificate name. Defaults to tls.crt.
	//
	// Note: This option is only used when TLSOpts does not set GetCertificate.
	// Note: If certificate or key doesn't exist a self-signed certificate will be used.
	MetricsCertName string `json:"metricsCertName" yaml:"metricsCertName"`

	// KeyName is the server key name. Defaults to tls.key.
	//
	// Note: This option is only used when TLSOpts does not set GetCertificate.
	// Note: If certificate or key doesn't exist a self-signed certificate will be used.
	MetricsKeyName string `json:"metricsKeyName" yaml:"metricsKeyName"`

	// HealthProbeBindAddress is the TCP address that the controller should bind to
	// for serving health probes
	// It can be set to "0" or "" to disable serving the health probe.
	HealthProbeBindAddress string `json:"healthProbeBindAddress" yaml:"healthProbeBindAddress"`

	// Readiness probe endpoint name, defaults to "readyz"
	ReadinessEndpointName string `json:"readinessEndpointName" yaml:"readinessEndpointName"`

	// Liveness probe endpoint name, defaults to "healthz"
	LivenessEndpointName string `json:"livenessEndpointName" yaml:"livenessEndpointName"`

	// PprofBindAddress is the TCP address that the controller should bind to
	// for serving pprof.
	// It can be set to "" or "0" to disable the pprof serving.
	// Since pprof may contain sensitive information, make sure to protect it
	// before exposing it to public.
	PprofBindAddress string `json:"pprofBindAddress" yaml:"pprofBindAddress"`

	// GracefulShutdownTimeout is the duration given to runnable and to stop before the manager actually returns on stop.
	// To disable graceful shutdown, set to time.Duration(0)
	// To use graceful shutdown without timeout, set to a negative duration, e.G. time.Duration(-1)
	// The graceful shutdown is skipped for safety reasons in case the leader election lease is lost.
	GracefulShutdownTimeout time.Duration `json:"gracefulShutdownTimeout" yaml:"gracefulShutdownTimeout"`
}

// SetDefaults set default values for manager options.
func (o *ManagerOptions) SetDefaults() {
	o.LeaderElectionID = utils.EmptyOr(o.LeaderElectionID, defaultLeaderElectionID)
	o.LeaderElectionResourceLock = utils.EmptyOr(o.LeaderElectionResourceLock, defaultLeaderElectionResourceLock)
	o.LeaseDuration = utils.EmptyOr(o.LeaseDuration, defaultLeaseDuration)
	o.RenewDeadline = utils.EmptyOr(o.RenewDeadline, defaultRenewDeadline)
	o.RetryPeriod = utils.EmptyOr(o.RetryPeriod, defaultRetryPeriod)
	o.MetricsBindAddress = utils.EmptyOr(o.MetricsBindAddress, ":8080")
	o.ReadinessEndpointName = utils.EmptyOr(o.ReadinessEndpointName, defaultReadinessEndpoint)
	o.LivenessEndpointName = utils.EmptyOr(o.LivenessEndpointName, defaultLivenessEndpoint)
	o.GracefulShutdownTimeout = utils.EmptyOr(o.GracefulShutdownTimeout, defaultGracefulShutdownPeriod)
}

// Validate validates the frpc service options.
func (o *ManagerOptions) Validate() (err error) {
	if o.LeaderElectionID == "" {
		err = errors.Join(err, fmt.Errorf("leaderElectionID is required"))
	}
	if o.LeaseDuration == 0 {
		err = errors.Join(err, fmt.Errorf("leaseDuration is required"))
	}
	if o.LeaderElectionResourceLock == "" {
		err = errors.Join(err, fmt.Errorf("leaderElectionResourceLock is required"))
	}
	if o.RenewDeadline == 0 {
		err = errors.Join(err, fmt.Errorf("renewDeadline is required"))
	}
	if o.RetryPeriod == 0 {
		err = errors.Join(err, fmt.Errorf("retryPeriod is required"))
	}
	if o.MetricsBindAddress == "" {
		err = errors.Join(err, fmt.Errorf("metricsBindAddress is required"))
	}
	if o.ReadinessEndpointName == "" {
		err = errors.Join(err, fmt.Errorf("readinessEndpointName is required"))
	}
	if o.LivenessEndpointName == "" {
		err = errors.Join(err, fmt.Errorf("livenessEndpointName is required"))
	}
	if o.GracefulShutdownTimeout == 0 {
		err = errors.Join(err, fmt.Errorf("gracefulShutdownTimeout is required"))
	}
	return err
}

// AddFlags add related command line parameters
func (o *ManagerOptions) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&o.LeaderElection, "manager.leader-election", o.LeaderElection,
		"Determines whether to use leader election when starting the manager.")

	fs.StringVar(&o.LeaderElectionResourceLock, "manager.leader-election-resource-lock", o.LeaderElectionResourceLock,
		"Determines which resource lock to use for leader election.")

	fs.StringVar(&o.LeaderElectionNamespace, "manager.leader-election-namespace", o.LeaderElectionNamespace,
		"Determines the namespace in which to run the leader election.")

	fs.StringVar(&o.LeaderElectionID, "manager.leader-election-id", o.LeaderElectionID,
		"Determines the name of the resource that leader election will use for holding the leader lock.")

	fs.BoolVar(&o.LeaderElectionReleaseOnCancel, "manager.leader-election-release-on-cancel", o.LeaderElectionReleaseOnCancel,
		"Defines if the leader should step down voluntarily when the Manager ends. This requires the binary"+
			" to immediately end when the Manager is stopped, otherwise this setting is unsafe. Setting this significantly "+
			"speeds up voluntary leader transitions as the new leader doesn't have to wait LeaseDuration time first")

	fs.DurationVar(&o.LeaseDuration, "manager.leader-election-lease-duration", o.LeaseDuration, "Is the duration "+
		"that non-leader candidates will wait to force acquire leadership. This is measured against time of last observed ack")

	fs.DurationVar(&o.RenewDeadline, "manager.leader-election-renew-deadline", o.RenewDeadline, "Is the duration "+
		"that the acting controlPlane will retry refreshing leadership before giving up")

	fs.DurationVar(&o.RetryPeriod, "manager.leader-election-retry-period", o.RetryPeriod, "Is the duration"+
		" the LeaderElector clients should wait between tries of actions")

	fs.StringVar(&o.HealthProbeBindAddress, "manager.health-probe-bind-address", o.HealthProbeBindAddress,
		"Is the TCP address that the controller should bind to for serving health probes It"+
			" can be set to \"0\" or \"\" to disable serving the health probe.")
}

func NewManagerOptions() *ManagerOptions {
	return &ManagerOptions{}
}
