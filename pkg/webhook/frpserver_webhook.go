package webhook

import (
	"context"
	"errors"
	"fmt"
	"github.com/fatedier/frp/pkg/util/util"
	"github.com/frp-sigs/frp-provisioner/pkg/api/v1beta1"
	"github.com/frp-sigs/frp-provisioner/pkg/utils"
	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	defaultServerPort              = 7000
	defaultProtocol                = "tcp"
	defaultDialServerTimeout       = 10
	defaultDialServerKeepAlive     = 7200
	defaultTCPMuxKeepaliveInterval = 60
	defaultHeartbeatInterval       = 30
	defaultHeartbeatTimeout        = 90
	defaultPoolCount               = 1
	defaultUDPPacketSize           = 1500
	defaultNatHoleSTUNServer       = "stun.easyvoip.com:3478"
)

var (
	supportedAuthMethods = []v1beta1.FrpServerAuthMethod{
		v1beta1.FrpServerAuthMethodToken,
		v1beta1.FrpServerAuthMethodOIDC,
	}
	supportedAuthScopes = []v1beta1.FrpServerAuthScope{
		v1beta1.FrpServerAuthScopeHeartBeats,
		v1beta1.FrpServerAuthScopeNewWorkConns,
	}
	supportedTransportProtocols = []v1beta1.FrpServerTransportProtocol{
		v1beta1.FrpServerTransportProtocolTCP,
		v1beta1.FrpServerTransportProtocolKCP,
		v1beta1.FrpServerTransportProtocolQUIC,
		v1beta1.FrpServerTransportProtocolWebsocket,
		v1beta1.FrpServerTransportProtocolWSS,
	}
)

type FrpServer struct {
	client client.Client
}

func (f *FrpServer) SetupWebhookWithManager(mgr ctrl.Manager) error {
	f.client = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1beta1.FrpServer{}).
		WithDefaulter(f).
		WithValidator(f).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-frp-gofrp-io-v1beta1-frpserver,mutating=true,failurePolicy=fail,sideEffects=None,groups=frp.gofrp.io,resources=frpservers,verbs=create;update,versions=v1beta1,name=mfrpserver.kb.io,admissionReviewVersions=v1

var _ admission.CustomDefaulter = &FrpServer{}

// Default implements admission.CustomDefaulter so a webhook will be registered for the type
func (f *FrpServer) Default(ctx context.Context, obj runtime.Object) error {
	r := obj.(*v1beta1.FrpServer)
	r.Spec.Auth.Method = utils.EmptyOr(r.Spec.Auth.Method, v1beta1.FrpServerAuthMethodToken)
	r.Spec.ServerPort = utils.EmptyOr(r.Spec.ServerPort, defaultServerPort)
	r.Spec.LoginFailExit = util.EmptyOr(r.Spec.LoginFailExit, lo.ToPtr(true))
	r.Spec.NatHoleSTUNServer = utils.EmptyOr(r.Spec.NatHoleSTUNServer, defaultNatHoleSTUNServer)
	r.Spec.UDPPacketSize = utils.EmptyOr(r.Spec.UDPPacketSize, defaultUDPPacketSize)
	// set Transport defaults
	r.Spec.Transport.Protocol = util.EmptyOr(r.Spec.Transport.Protocol, defaultProtocol)
	r.Spec.Transport.DialServerTimeout = util.EmptyOr(r.Spec.Transport.DialServerTimeout, defaultDialServerTimeout)
	r.Spec.Transport.DialServerKeepAlive = util.EmptyOr(r.Spec.Transport.DialServerKeepAlive, defaultDialServerKeepAlive)
	r.Spec.Transport.PoolCount = util.EmptyOr(r.Spec.Transport.PoolCount, defaultPoolCount)
	r.Spec.Transport.TCPMux = util.EmptyOr(r.Spec.Transport.TCPMux, lo.ToPtr(true))
	r.Spec.Transport.TCPMuxKeepaliveInterval = util.EmptyOr(r.Spec.Transport.TCPMuxKeepaliveInterval, defaultTCPMuxKeepaliveInterval)
	r.Spec.Transport.HeartbeatInterval = util.EmptyOr(r.Spec.Transport.HeartbeatInterval, defaultHeartbeatInterval)
	r.Spec.Transport.HeartbeatTimeout = util.EmptyOr(r.Spec.Transport.HeartbeatTimeout, defaultHeartbeatTimeout)
	r.Spec.Transport.TLS.Enable = util.EmptyOr(r.Spec.Transport.TLS.Enable, lo.ToPtr(true))
	r.Spec.Transport.TLS.DisableCustomTLSFirstByte = util.EmptyOr(r.Spec.Transport.TLS.DisableCustomTLSFirstByte, lo.ToPtr(true))
	if r.Spec.Transport.Protocol == v1beta1.FrpServerTransportProtocolQUIC {
		r.Spec.Transport.QUIC.KeepalivePeriod = util.EmptyOr(r.Spec.Transport.QUIC.KeepalivePeriod, 10)
		r.Spec.Transport.QUIC.MaxIdleTimeout = util.EmptyOr(r.Spec.Transport.QUIC.MaxIdleTimeout, 30)
		r.Spec.Transport.QUIC.MaxIncomingStreams = util.EmptyOr(r.Spec.Transport.QUIC.MaxIncomingStreams, 100000)
	}
	// set status defaults
	r.Status.Phase = utils.EmptyOr(r.Status.Phase, v1beta1.FrpServerPending)
	return nil
}

//+kubebuilder:webhook:path=/validate-frp-gofrp-io-v1beta1-frpserver,mutating=false,failurePolicy=fail,sideEffects=None,groups=frp.gofrp.io,resources=frpservers,verbs=create;update;delete,versions=v1beta1,name=vfrpserver.kb.io,admissionReviewVersions=v1

var _ admission.CustomValidator = &FrpServer{}

func (f *FrpServer) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	r := obj.(*v1beta1.FrpServer)
	if !lo.Contains(supportedAuthMethods, r.Spec.Auth.Method) {
		err = errors.Join(err, fmt.Errorf("invalid spec.auth.method, optional values are %+v", supportedAuthMethods))
	}
	if !lo.Every(supportedAuthScopes, r.Spec.Auth.AdditionalScopes) {
		err = errors.Join(err, fmt.Errorf("invalid spec.auth.additionalScopes, optional values are %v", supportedAuthScopes))
	}
	if r.Spec.Auth.Method == v1beta1.FrpServerAuthMethodToken && r.Spec.Auth.Token == "" {
		err = errors.Join(err, fmt.Errorf("field spec.auth.token should not be empty"))
	}
	if r.Spec.ServerAddr == "" {
		err = errors.Join(err, fmt.Errorf("field spec.serverAddr should not be empty"))
	}
	if r.Spec.ServerPort <= 0 {
		err = errors.Join(err, fmt.Errorf("field spec.serverPort should not be empty"))
	}
	if len(r.Spec.ExternalIPs) == 0 {
		err = errors.Join(err, fmt.Errorf("field spec.externalIPs should not be empty"))
	}
	if r.Spec.Transport.HeartbeatTimeout > 0 && r.Spec.Transport.HeartbeatInterval > 0 {
		if r.Spec.Transport.HeartbeatTimeout < r.Spec.Transport.HeartbeatInterval {
			err = errors.Join(err, fmt.Errorf("invalid spec.transport.heartbeatTimeout,"+
				" spec.transport.heartbeatTimeout should not less than spec.transport.heartbeatInterval"))
		}
	}
	if !lo.FromPtr(r.Spec.Transport.TLS.Enable) {
		if r.Spec.Transport.TLS.SecretRef != nil {
			warnings = append(warnings, fmt.Sprintf("field spec.transport.tls.secretRef should be empty when transport.tls.enable is false"))
		}

		if r.Spec.Transport.TLS.CertFileName != "" {
			warnings = append(warnings, fmt.Sprintf("field spec.transport.tls.certFileName should be empty when transport.tls.enable is false"))
		}
		if r.Spec.Transport.TLS.KeyFileName != "" {
			warnings = append(warnings, fmt.Sprintf("field spec.transport.tls.keyFileName should be empty when transport.tls.enable is false"))
		}
		if r.Spec.Transport.TLS.CaFileName != "" {
			warnings = append(warnings, fmt.Sprintf("field spec.transport.tls.caFileName should be empty when transport.tls.enable is false"))
		}
		r.Spec.Transport.TLS.CertFileName = ""
		r.Spec.Transport.TLS.KeyFileName = ""
		r.Spec.Transport.TLS.CaFileName = ""
		r.Spec.Transport.TLS.SecretRef = nil
	}
	if !lo.Contains(supportedTransportProtocols, r.Spec.Transport.Protocol) {
		err = errors.Join(err, fmt.Errorf("invalid spec.transport.protocol, optional values are %+v", supportedTransportProtocols))
	}
	return warnings, err
}

// ValidateUpdate implements admission.CustomValidator so a webhook will be registered for the type
func (f *FrpServer) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (warnings admission.Warnings, err error) {
	r := oldObj.(*v1beta1.FrpServer)
	if !lo.Contains(supportedAuthMethods, r.Spec.Auth.Method) {
		err = errors.Join(err, fmt.Errorf("invalid spec.auth.method, optional values are %+v", supportedAuthMethods))
	}
	if !lo.Every(supportedAuthScopes, r.Spec.Auth.AdditionalScopes) {
		err = errors.Join(err, fmt.Errorf("invalid spec.auth.additionalScopes, optional values are %v", supportedAuthScopes))
	}
	if r.Spec.Auth.Method == v1beta1.FrpServerAuthMethodToken && r.Spec.Auth.Token == "" {
		err = errors.Join(err, fmt.Errorf("field spec.auth.token should not be empty"))
	}
	if r.Spec.ServerAddr == "" {
		err = errors.Join(err, fmt.Errorf("field spec.serverAddr should not be empty"))
	}
	if r.Spec.ServerPort <= 0 {
		err = errors.Join(err, fmt.Errorf("field spec.serverPort should not be empty"))
	}
	if len(r.Spec.ExternalIPs) == 0 {
		err = errors.Join(err, fmt.Errorf("field spec.externalIPs should not be empty"))
	}
	if r.Spec.Transport.HeartbeatTimeout > 0 && r.Spec.Transport.HeartbeatInterval > 0 {
		if r.Spec.Transport.HeartbeatTimeout < r.Spec.Transport.HeartbeatInterval {
			err = errors.Join(err, fmt.Errorf("invalid spec.transport.heartbeatTimeout,"+
				" spec.transport.heartbeatTimeout should not less than spec.transport.heartbeatInterval"))
		}
	}
	if !lo.FromPtr(r.Spec.Transport.TLS.Enable) {
		if r.Spec.Transport.TLS.SecretRef != nil {
			warnings = append(warnings, fmt.Sprintf("field spec.transport.tls.secretRef should be empty when transport.tls.enable is false"))
		}

		if r.Spec.Transport.TLS.CertFileName != "" {
			warnings = append(warnings, fmt.Sprintf("field spec.transport.tls.certFileName should be empty when transport.tls.enable is false"))
		}
		if r.Spec.Transport.TLS.KeyFileName != "" {
			warnings = append(warnings, fmt.Sprintf("field spec.transport.tls.keyFileName should be empty when transport.tls.enable is false"))
		}
		if r.Spec.Transport.TLS.CaFileName != "" {
			warnings = append(warnings, fmt.Sprintf("field spec.transport.tls.caFileName should be empty when transport.tls.enable is false"))
		}
		r.Spec.Transport.TLS.CertFileName = ""
		r.Spec.Transport.TLS.KeyFileName = ""
		r.Spec.Transport.TLS.CaFileName = ""
		r.Spec.Transport.TLS.SecretRef = nil
	}
	if !lo.Contains(supportedTransportProtocols, r.Spec.Transport.Protocol) {
		err = errors.Join(err, fmt.Errorf("invalid spec.transport.protocol, optional values are %+v", supportedTransportProtocols))
	}
	return warnings, err
}

// ValidateDelete implements admission.CustomValidator so a webhook will be registered for the type
func (f *FrpServer) ValidateDelete(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return nil, err
}
