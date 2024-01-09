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

package config

import (
	"errors"
	"fmt"
	"github.com/fatedier/frp/pkg/util/util"
	"github.com/spf13/pflag"
	"k8s.io/api/core/v1"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"time"
)

const (
	defaultLeaderElectionResourceLock = "leases"
	defaultLeaderElectionID           = "frp-provisioner"
	defaultLeaseDuration              = 15 * time.Second
	defaultRenewDeadline              = 10 * time.Second
	defaultRetryPeriod                = 2 * time.Second
	defaultGracefulShutdownPeriod     = 30 * time.Second
	defaultWebhookBindAddress         = ":9443"
	defaultWebhookCertName            = "tls.crt"
	defaultWebhookKeyName             = "tls.key"
)

const defaultPodTemplate = `
metadata:
 labels:
   app: frp-client
 name: frp-client
spec:
 containers:
 - image: busybox:latest
   imagePullPolicy: Always
   name: frp-client
   resources:
     limits:
       cpu: 500m
       memory: 1Gi
     requests:
       cpu: 100m
       memory: 128Mi
   command:
   - tail
   - "-f"
`

// ManagerOptions  contains the configuration for the manager.
type ManagerOptions struct {
	// LeaderElection determines whether to use leader election when
	// starting the manager.
	LeaderElection bool `json:"leaderElection"`

	// LeaderElectionResourceLock determines which resource lock to use for leader election,
	// defaults to "leases". Change this value only if you know what you are doing.
	LeaderElectionResourceLock string `json:"leaderElectionResourceLock"`

	// LeaderElectionNamespace determines the namespace in which the leader
	// election resource will be created.
	LeaderElectionNamespace string `json:"leaderElectionNamespace"`

	// LeaderElectionID determines the name of the resource that leader election
	// will use for holding the leader lock.
	LeaderElectionID string `json:"leaderElectionID"`

	// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
	// when the Manager ends. This requires the binary to immediately end when the
	// Manager is stopped, otherwise this setting is unsafe. Setting this significantly
	// speeds up voluntary leader transitions as the new leader doesn't have to wait
	// LeaseDuration time first.
	LeaderElectionReleaseOnCancel bool `json:"leaderElectionReleaseOnCancel"`

	// LeaseDuration is the duration that non-leader candidates will
	// wait to force acquire leadership. This is measured against time of
	// last observed ack. Default is 15 seconds.
	LeaseDuration time.Duration `json:"leaseDuration"`

	// RenewDeadline is the duration that the acting controlPlane will retry
	// refreshing leadership before giving up. Default is 10 seconds.
	RenewDeadline time.Duration `json:"renewDeadline"`

	// RetryPeriod is the duration the LeaderElector clients should wait
	// between tries of actions. Default is 2 seconds.
	RetryPeriod time.Duration `json:"retryPeriod"`

	// SecureServing enables serving metrics via https.
	// Per default metrics will be served via http.
	MetricsSecureServing bool `json:"metricsSecureServing"`

	// BindAddress is the bind address for the metrics server.
	// It will be defaulted to ":8080" if unspecified.
	// Set this to "0" to disable the metrics server.
	MetricsBindAddress string `json:"metricsBindAddress"`

	// CertDir is the directory that contains the server key and certificate. Defaults to
	// <temp-dir>/k8s-metrics-server/serving-certs.
	//
	// Note: This option is only used when TLSOpts does not set GetCertificate.
	// Note: If certificate or key doesn't exist a self-signed certificate will be used.
	MetricsCertDir string `json:"metricsCertDir"`

	// CertName is the server certificate name. Defaults to tls.crt.
	//
	// Note: This option is only used when TLSOpts does not set GetCertificate.
	// Note: If certificate or key doesn't exist a self-signed certificate will be used.
	MetricsCertName string `json:"metricsCertName"`

	// KeyName is the server key name. Defaults to tls.key.
	//
	// Note: This option is only used when TLSOpts does not set GetCertificate.
	// Note: If certificate or key doesn't exist a self-signed certificate will be used.
	MetricsKeyName string `json:"metricsKeyName"`

	// WebhookBindAddress is the address that the server will listen on.
	// Defaults to "" - all addresses.
	WebhookBindAddress string `json:"webhookBindAddress"`

	// CertDir is the directory that contains the server key and certificate. Defaults to
	// <temp-dir>/k8s-webhook-server/serving-certs.
	WebhookCertDir string `json:"webhookCertDir"`

	// CertName is the server certificate name. Defaults to tls.crt.
	WebhookCertName string `json:"webhookCertName"`

	// KeyName is the server key name. Defaults to tls.key.
	//
	// Note: This option is only used when TLSOpts does not set GetCertificate.
	WebhookKeyName string `json:"webhookKeyName"`

	// ClientCAName is the CA certificate name which server used to verify remote(client)'s certificate.
	// Defaults to "", which means server does not verify client's certificate.
	WebhookClientCAName string `json:"webhookClientCAName"`

	// HealthProbeBindAddress is the TCP address that the controller should bind to
	// for serving health probes
	// It can be set to "0" or "" to disable serving the health probe.
	HealthProbeBindAddress string `json:"healthProbeBindAddress"`

	// PprofBindAddress is the TCP address that the controller should bind to
	// for serving pprof.
	// It can be set to "" or "0" to disable the pprof serving.
	// Since pprof may contain sensitive information, make sure to protect it
	// before exposing it to public.
	PprofBindAddress string `json:"pprofBindAddress"`

	// GracefulShutdownTimeout is the duration given to runnable and to stop before the manager actually returns on stop.
	// To disable graceful shutdown, set to time.Duration(0)
	// To use graceful shutdown without timeout, set to a negative duration, e.G. time.Duration(-1)
	// The graceful shutdown is skipped for safety reasons in case the leader election lease is lost.
	GracefulShutdownTimeout time.Duration `json:"gracefulShutdownTimeout"`

	// PodTemplate The path to the pod template file for the FRP client, which will be used to generate pods
	PodTemplate string `json:"PodTemplate"`
}

// SetDefaults set default values for manager options.
func (o *ManagerOptions) SetDefaults() {
	o.LeaderElectionID = util.EmptyOr(o.LeaderElectionID, defaultLeaderElectionID)

	o.LeaderElectionResourceLock = util.EmptyOr(o.LeaderElectionResourceLock, defaultLeaderElectionResourceLock)

	o.LeaseDuration = util.EmptyOr(o.LeaseDuration, defaultLeaseDuration)

	o.RenewDeadline = util.EmptyOr(o.RenewDeadline, defaultRenewDeadline)

	o.RetryPeriod = util.EmptyOr(o.RetryPeriod, defaultRetryPeriod)

	o.MetricsBindAddress = util.EmptyOr(o.MetricsBindAddress, ":8080")

	o.GracefulShutdownTimeout = util.EmptyOr(o.GracefulShutdownTimeout, defaultGracefulShutdownPeriod)

	o.WebhookBindAddress = util.EmptyOr(o.WebhookBindAddress, defaultWebhookBindAddress)

	o.WebhookCertDir = util.EmptyOr(o.WebhookCertDir, filepath.Join(os.TempDir(), "k8s-webhook-server", "serving-certs"))

	o.WebhookCertName = util.EmptyOr(o.WebhookCertName, defaultWebhookCertName)

	o.WebhookKeyName = util.EmptyOr(o.WebhookKeyName, defaultWebhookKeyName)

	o.PodTemplate = util.EmptyOr(o.PodTemplate, defaultPodTemplate)

	o.MetricsCertDir = util.EmptyOr(o.MetricsCertDir, filepath.Join(os.TempDir(), "k8s-metrics-server", "serving-certs"))
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

	if o.GracefulShutdownTimeout == 0 {
		err = errors.Join(err, fmt.Errorf("gracefulShutdownTimeout is required"))
	}

	if o.PodTemplate == "" {
		err = errors.Join(err, fmt.Errorf("PodTemplate is required"))
	}
	p := v1.Pod{}
	err = yaml.Unmarshal([]byte(o.PodTemplate), &p)
	if err != nil {
		err = errors.Join(err, fmt.Errorf("unable parse podTemplate with yaml: %v", o.PodTemplate))
	} else if len(p.Spec.Containers) == 0 {
		err = errors.Join(err, fmt.Errorf("podTemplate does not specify any container"))
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

	fs.BoolVar(&o.MetricsSecureServing, "manager.MetricsSecureServing", o.MetricsSecureServing, "")

	fs.StringVar(&o.MetricsBindAddress, "manager.metrics-bind-address", o.MetricsBindAddress, "The address the metric endpoint binds to.")

	//fs.StringVar(&o.PodTemplate, "manager.pod-template-file", o.PodTemplate, "The path to the pod template file for the FRP client, which will be used to generate pods.")

	fs.StringVar(&o.PprofBindAddress, "manager.pprof-bind-address", o.PprofBindAddress, "Is the tcp address that the controller should bind to "+
		"for serving pprof. It can be set to \"\" or \"0\" to disable the pprof serving.")

	fs.StringVar(&o.WebhookClientCAName, "manager.webhook-client-ca-name", o.WebhookClientCAName, "Is the CA certificate name which server used to verify remote(client)'s certificate."+
		" Defaults to \"\", which means server does not verify client's certificate.")

	fs.StringVar(&o.WebhookKeyName, "manager.webhook-key-name", o.WebhookKeyName, "Is the webhook server tls key filename.")

	fs.StringVar(&o.WebhookCertName, "manager.webhook-cert-name", o.WebhookCertName, "Is the webhook server tls certificate filename.")

	fs.StringVar(&o.WebhookCertDir, "manager.webhook-cert-dir", o.WebhookCertDir, "Is the directory that contains the webhook server key and certificate")

	fs.StringVar(&o.WebhookBindAddress, "manager.webhook-bind-address", o.WebhookBindAddress, "Is the address that the webhook server will listen on")

	fs.StringVar(&o.MetricsKeyName, "manager.metrics-key-name", o.MetricsKeyName, "Is the metrics server tls key filename.")

	fs.StringVar(&o.MetricsCertDir, "manager.metrics-cert-dir", o.MetricsCertDir, "Is the directory that contains the metrics server key and certificate")

	fs.StringVar(&o.MetricsCertName, "manager.metrics-cert-name", o.MetricsCertName, "Is the metrics server tls certificate filename.")

	fs.DurationVar(&o.GracefulShutdownTimeout, "manager.graceful-shutdown-timeout", o.GracefulShutdownTimeout, "is the duration given to runnable and to stop before the manager actually returns on stop."+
		" To disable graceful shutdown, set to 0, To use graceful shutdown without timeout, set to a negative duration, eg: -1, The graceful shutdown is skipped for safety reasons in case the leader election lease is lost.")
}
