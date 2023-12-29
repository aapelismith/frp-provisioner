package validator

import (
	"context"
	"errors"
	"fmt"
	"github.com/frp-sigs/frp-provisioner/pkg/utils/frp"
	"github.com/frp-sigs/frp-provisioner/pkg/utils/validate"
	errors2 "k8s.io/apimachinery/pkg/api/errors"

	"github.com/fatedier/frp/pkg/util/util"
	"github.com/frp-sigs/frp-provisioner/pkg/api/v1beta1"
	"github.com/samber/lo"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type FrpServerValidator struct {
	client.Client
}

func (f *FrpServerValidator) SetupWebhookWithManager(mgr ctrl.Manager) error {
	f.Client = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1beta1.FrpServer{}).
		WithDefaulter(f).
		WithValidator(f).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-frp-gofrp-io-v1beta1-frpserver,mutating=true,failurePolicy=fail,sideEffects=None,groups=frp.gofrp.io,resources=frpservers,verbs=create;update,versions=v1beta1,name=mfrpserver.kb.io,admissionReviewVersions=v1
var _ admission.CustomDefaulter = &FrpServerValidator{}

// Default implements admission.CustomDefaulter so a webhook will be registered for the type
func (f *FrpServerValidator) Default(ctx context.Context, obj runtime.Object) error {
	r := obj.(*v1beta1.FrpServer)
	r.Spec.Auth.Method = util.EmptyOr(r.Spec.Auth.Method, v1beta1.FrpServerAuthMethodToken)
	r.Spec.ServerPort = util.EmptyOr(r.Spec.ServerPort, 7000)
	r.Spec.LoginFailExit = util.EmptyOr(r.Spec.LoginFailExit, lo.ToPtr(true))
	r.Spec.NatHoleSTUNServer = util.EmptyOr(r.Spec.NatHoleSTUNServer, defaultNatHoleSTUNServer)
	r.Spec.UDPPacketSize = util.EmptyOr(r.Spec.UDPPacketSize, 1500)
	// set Transport defaults
	r.Spec.Transport.Protocol = util.EmptyOr(r.Spec.Transport.Protocol, v1beta1.FrpServerTransportProtocolTCP)
	r.Spec.Transport.DialServerTimeout = util.EmptyOr(r.Spec.Transport.DialServerTimeout, 10)
	r.Spec.Transport.DialServerKeepAlive = util.EmptyOr(r.Spec.Transport.DialServerKeepAlive, 7200)
	r.Spec.Transport.PoolCount = util.EmptyOr(r.Spec.Transport.PoolCount, 1)
	r.Spec.Transport.TCPMux = util.EmptyOr(r.Spec.Transport.TCPMux, lo.ToPtr(true))
	r.Spec.Transport.TCPMuxKeepaliveInterval = util.EmptyOr(r.Spec.Transport.TCPMuxKeepaliveInterval, 60)
	r.Spec.Transport.HeartbeatInterval = util.EmptyOr(r.Spec.Transport.HeartbeatInterval, 30)
	r.Spec.Transport.HeartbeatTimeout = util.EmptyOr(r.Spec.Transport.HeartbeatTimeout, 90)
	r.Spec.Transport.TLS.Enable = util.EmptyOr(r.Spec.Transport.TLS.Enable, lo.ToPtr(true))
	r.Spec.Transport.TLS.DisableCustomTLSFirstByte = util.EmptyOr(r.Spec.Transport.TLS.DisableCustomTLSFirstByte, lo.ToPtr(true))
	if r.Spec.Transport.Protocol == v1beta1.FrpServerTransportProtocolQUIC {
		r.Spec.Transport.QUIC.KeepalivePeriod = util.EmptyOr(r.Spec.Transport.QUIC.KeepalivePeriod, 10)
		r.Spec.Transport.QUIC.MaxIdleTimeout = util.EmptyOr(r.Spec.Transport.QUIC.MaxIdleTimeout, 30)
		r.Spec.Transport.QUIC.MaxIncomingStreams = util.EmptyOr(r.Spec.Transport.QUIC.MaxIncomingStreams, 100000)
	}
	transportTLSEnable := lo.FromPtr(r.Spec.Transport.TLS.Enable)
	if !transportTLSEnable {
		r.Spec.Transport.TLS.SecretRef = nil
		r.Spec.Transport.TLS.CaFileName = ""
		r.Spec.Transport.TLS.CertFileName = ""
		r.Spec.Transport.TLS.KeyFileName = ""
	}
	if transportTLSEnable && r.Spec.Transport.TLS.SecretRef != nil {
		r.Spec.Transport.TLS.CaFileName = util.EmptyOr(r.Spec.Transport.TLS.CaFileName, "tls.ca")
		r.Spec.Transport.TLS.CertFileName = util.EmptyOr(r.Spec.Transport.TLS.CertFileName, "tls.crt")
		r.Spec.Transport.TLS.KeyFileName = util.EmptyOr(r.Spec.Transport.TLS.KeyFileName, "tls.key")
	}
	return ctx.Err()
}

// +kubebuilder:webhook:path=/validate-frp-gofrp-io-v1beta1-frpserver,mutating=false,failurePolicy=fail,sideEffects=None,groups=frp.gofrp.io,resources=frpservers,verbs=create;update;delete,versions=v1beta1,name=vfrpserver.kb.io,admissionReviewVersions=v1
var _ admission.CustomValidator = &FrpServerValidator{}

func (f *FrpServerValidator) ValidateCreate(ctx context.Context, object runtime.Object) (warnings admission.Warnings, errs error) {
	obj := object.(*v1beta1.FrpServer)
	if !lo.Contains(authMethods, obj.Spec.Auth.Method) {
		errs = errors.Join(errs, fmt.Errorf("invalid spec.auth.method, optional values are %+v", authMethods))
	}
	if !lo.Every(authScopes, obj.Spec.Auth.AdditionalScopes) {
		errs = errors.Join(errs, fmt.Errorf("invalid spec.auth.authScopes, optional values are %v", authScopes))
	}
	if obj.Spec.Auth.Method == v1beta1.FrpServerAuthMethodToken && obj.Spec.Auth.Token == "" {
		errs = errors.Join(errs, fmt.Errorf("field spec.auth.token should not be empty"))
	}
	if obj.Spec.ServerAddr == "" {
		errs = errors.Join(errs, fmt.Errorf("field spec.serverAddr should not be empty"))
	}
	if err := validate.ValidatePort(obj.Spec.ServerPort); err != nil {
		errs = errors.Join(errs, fmt.Errorf("invalid field spec.serverPort, got: %w", err))
	}
	if len(obj.Spec.ExternalIPs) == 0 {
		errs = errors.Join(errs, fmt.Errorf("field spec.externalIPs should not be empty"))
	}
	if obj.Spec.Transport.HeartbeatTimeout > 0 && obj.Spec.Transport.HeartbeatInterval > 0 {
		if obj.Spec.Transport.HeartbeatTimeout < obj.Spec.Transport.HeartbeatInterval {
			errs = errors.Join(errs, fmt.Errorf("invalid spec.transport.heartbeatTimeout,"+
				" spec.transport.heartbeatTimeout should not less than spec.transport.heartbeatInterval"))
		}
	}
	transportTLSEnable := lo.FromPtr(obj.Spec.Transport.TLS.Enable)
	if !transportTLSEnable {
		if obj.Spec.Transport.TLS.SecretRef != nil {
			warnings = append(warnings, fmt.Sprintf("field spec.transport.tls.secretRef should be empty when transport.tls.enable is false"))
		}
		if obj.Spec.Transport.TLS.CertFileName != "" {
			warnings = append(warnings, fmt.Sprintf("field spec.transport.tls.certFileName should be empty when transport.tls.enable is false"))
		}
		if obj.Spec.Transport.TLS.KeyFileName != "" {
			warnings = append(warnings, fmt.Sprintf("field spec.transport.tls.keyFileName should be empty when transport.tls.enable is false"))
		}
		if obj.Spec.Transport.TLS.CaFileName != "" {
			warnings = append(warnings, fmt.Sprintf("field spec.transport.tls.caFileName should be empty when transport.tls.enable is false"))
		}
	}
	if obj.Spec.Transport.TLS.SecretRef != nil && transportTLSEnable {
		if obj.Spec.Transport.TLS.SecretRef.Name != "" && obj.Spec.Transport.TLS.SecretRef.Namespace == "" {
			errs = errors.Join(errs, fmt.Errorf("field spec.transport.tls.secretRef.namespace"+
				" should not be empty when spec.transport.tls.secretRef.name is not empty"))
		}
		if obj.Spec.Transport.TLS.SecretRef.Name == "" && obj.Spec.Transport.TLS.SecretRef.Namespace != "" {
			errs = errors.Join(errs, fmt.Errorf("field spec.transport.tls.secretRef.name"+
				" should not be empty when spec.transport.tls.secretRef.namespace is not empty"))
		}
		if !lo.Contains(transportProtocols, obj.Spec.Transport.Protocol) {
			errs = errors.Join(errs, fmt.Errorf("invalid spec.transport.protocol, optional values are %+v", transportProtocols))
		}
		if obj.Spec.Transport.TLS.SecretRef.Name != "" && obj.Spec.Transport.TLS.SecretRef.Namespace != "" {
			objKey := client.ObjectKey{
				Name:      obj.Spec.Transport.TLS.SecretRef.Name,
				Namespace: obj.Spec.Transport.TLS.SecretRef.Namespace,
			}
			secretObj := &v1.Secret{}
			if err := f.Get(ctx, objKey, secretObj); err != nil {
				errs = errors.Join(errs, fmt.Errorf("unable get secret '%s/%s', got: %w", objKey.Namespace, objKey.Name, err))
			}
			if _, ok := secretObj.Data[obj.Spec.Transport.TLS.CaFileName]; !ok {
				errs = errors.Join(errs, fmt.Errorf("file '%s' not found on secret '%s/%s'", obj.Spec.Transport.TLS.CaFileName, objKey.Namespace, objKey.Name))
			}
			if _, ok := secretObj.Data[obj.Spec.Transport.TLS.KeyFileName]; !ok {
				errs = errors.Join(errs, fmt.Errorf("file '%s' not found on secret '%s/%s'", obj.Spec.Transport.TLS.KeyFileName, objKey.Namespace, objKey.Name))
			}
			if _, ok := secretObj.Data[obj.Spec.Transport.TLS.CertFileName]; !ok {
				errs = errors.Join(errs, fmt.Errorf("file '%s' not found on secret '%s/%s'", obj.Spec.Transport.TLS.CertFileName, objKey.Namespace, objKey.Name))
			}
		}
	}
	if errs == nil {
		commonConfig, err := frp.GenClientCommonConfig(ctx, f.Client, obj)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("unable generate frp config, got: %w", err))
		} else {
			err = validate.ValidateClientCommonConfig(ctx, commonConfig)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to validate frp config, got: %w", err))
			}
		}
	}
	return warnings, errs
}

// ValidateUpdate implements admission.CustomValidator so a webhook will be registered for the type
func (f *FrpServerValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (warnings admission.Warnings, errs error) {
	obj := newObj.(*v1beta1.FrpServer)
	if !lo.Contains(authMethods, obj.Spec.Auth.Method) {
		errs = errors.Join(errs, fmt.Errorf("invalid spec.auth.method, optional values are %+v", authMethods))
	}
	if !lo.Every(authScopes, obj.Spec.Auth.AdditionalScopes) {
		errs = errors.Join(errs, fmt.Errorf("invalid spec.auth.authScopes, optional values are %v", authScopes))
	}
	if obj.Spec.Auth.Method == v1beta1.FrpServerAuthMethodToken && obj.Spec.Auth.Token == "" {
		errs = errors.Join(errs, fmt.Errorf("field spec.auth.token should not be empty"))
	}
	if obj.Spec.ServerAddr == "" {
		errs = errors.Join(errs, fmt.Errorf("field spec.serverAddr should not be empty"))
	}
	if obj.Spec.ServerPort <= 0 {
		errs = errors.Join(errs, fmt.Errorf("field spec.serverPort should not be empty"))
	}
	if len(obj.Spec.ExternalIPs) == 0 {
		errs = errors.Join(errs, fmt.Errorf("field spec.externalIPs should not be empty"))
	}
	if obj.Spec.Transport.HeartbeatTimeout > 0 && obj.Spec.Transport.HeartbeatInterval > 0 {
		if obj.Spec.Transport.HeartbeatTimeout < obj.Spec.Transport.HeartbeatInterval {
			errs = errors.Join(errs, fmt.Errorf("invalid spec.transport.heartbeatTimeout,"+
				" spec.transport.heartbeatTimeout should not less than spec.transport.heartbeatInterval"))
		}
	}
	transportTLSEnable := lo.FromPtr(obj.Spec.Transport.TLS.Enable)
	if !transportTLSEnable {
		if obj.Spec.Transport.TLS.SecretRef != nil {
			warnings = append(warnings, fmt.Sprintf("field spec.transport.tls.secretRef should be empty when transport.tls.enable is false"))
		}

		if obj.Spec.Transport.TLS.CertFileName != "" {
			warnings = append(warnings, fmt.Sprintf("field spec.transport.tls.certFileName should be empty when transport.tls.enable is false"))
		}
		if obj.Spec.Transport.TLS.KeyFileName != "" {
			warnings = append(warnings, fmt.Sprintf("field spec.transport.tls.keyFileName should be empty when transport.tls.enable is false"))
		}
		if obj.Spec.Transport.TLS.CaFileName != "" {
			warnings = append(warnings, fmt.Sprintf("field spec.transport.tls.caFileName should be empty when transport.tls.enable is false"))
		}
	}
	if obj.Spec.Transport.TLS.SecretRef != nil && transportTLSEnable {
		if obj.Spec.Transport.TLS.SecretRef.Name != "" && obj.Spec.Transport.TLS.SecretRef.Namespace == "" {
			errs = errors.Join(errs, fmt.Errorf("field spec.transport.tls.secretRef.namespace"+
				" should not be empty when spec.transport.tls.secretRef.name is not empty"))
		}
		if obj.Spec.Transport.TLS.SecretRef.Name == "" && obj.Spec.Transport.TLS.SecretRef.Namespace != "" {
			errs = errors.Join(errs, fmt.Errorf("field spec.transport.tls.secretRef.name"+
				" should not be empty when spec.transport.tls.secretRef.namespace is not empty"))
		}
		if !lo.Contains(transportProtocols, obj.Spec.Transport.Protocol) {
			errs = errors.Join(errs, fmt.Errorf("invalid spec.transport.protocol, optional values are %+v", transportProtocols))
		}
		if obj.Spec.Transport.TLS.SecretRef.Name != "" && obj.Spec.Transport.TLS.SecretRef.Namespace != "" {
			objKey := client.ObjectKey{
				Name:      obj.Spec.Transport.TLS.SecretRef.Name,
				Namespace: obj.Spec.Transport.TLS.SecretRef.Namespace,
			}
			secretObj := &v1.Secret{}
			if err := f.Get(ctx, objKey, secretObj); err != nil {
				errs = errors.Join(errs, fmt.Errorf("unable get secret '%s/%s', got: %w", objKey.Namespace, objKey.Name, err))
			}
			if _, ok := secretObj.Data[obj.Spec.Transport.TLS.CaFileName]; !ok {
				errs = errors.Join(errs, fmt.Errorf("file '%s' not found on secret '%s/%s'", obj.Spec.Transport.TLS.CaFileName, objKey.Namespace, objKey.Name))
			}
			if _, ok := secretObj.Data[obj.Spec.Transport.TLS.KeyFileName]; !ok {
				errs = errors.Join(errs, fmt.Errorf("file '%s' not found on secret '%s/%s'", obj.Spec.Transport.TLS.KeyFileName, objKey.Namespace, objKey.Name))
			}
			if _, ok := secretObj.Data[obj.Spec.Transport.TLS.CertFileName]; !ok {
				errs = errors.Join(errs, fmt.Errorf("file '%s' not found on secret '%s/%s'", obj.Spec.Transport.TLS.CertFileName, objKey.Namespace, objKey.Name))
			}
		}
	}
	if errs == nil {
		commonConfig, err := frp.GenClientCommonConfig(ctx, f.Client, obj)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("unable generate frp config, got: %w", err))
		} else {
			err = validate.ValidateClientCommonConfig(ctx, commonConfig)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to validate frp config, got: %w", err))
			}
		}
	}
	return warnings, errs
}

// ValidateDelete implements admission.CustomValidator so a webhook will be registered for the type
func (f *FrpServerValidator) ValidateDelete(ctx context.Context, object runtime.Object) (warnings admission.Warnings, err error) {
	obj := object.(*v1beta1.FrpServer)
	for _, reference := range obj.Status.ServiceReferences {
		svc := &v1.Service{}
		objKey := client.ObjectKey{
			Name:      reference.Name,
			Namespace: reference.Namespace,
		}
		if objKey.Name == "" || objKey.Namespace == "" {
			continue
		}
		if err := f.Get(ctx, objKey, svc); err != nil {
			if errors2.IsNotFound(err) {
				continue
			}
			return nil, fmt.Errorf("unable get service '%s', got: %w", objKey, err)
		}
		if len(svc.Labels) != 0 && svc.Labels[v1beta1.AnnotationFrpServerNameKey] == obj.GetName() {

		}
	}
	return nil, err
}
