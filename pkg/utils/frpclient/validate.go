package frpclient

import (
	"context"
	"fmt"
	frpclient "github.com/fatedier/frp/client"
	"github.com/fatedier/frp/pkg/auth"
	configv1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/util/version"
	"github.com/frp-sigs/frp-provisioner/pkg/api/v1beta1"
	"github.com/samber/lo"
	v1 "k8s.io/api/core/v1"
	"os"
	"runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

// ValidatePort checks that the network port is in range
func ValidatePort(port int) error {
	if 0 <= port && port <= 65535 {
		return nil
	}
	return fmt.Errorf("port number %d must be in the range 0..65535", port)
}

// ValidateFrpServerConfig validate and check config from v1beta1.FrpServer
func ValidateFrpServerConfig(ctx context.Context, cli client.Client, obj *v1beta1.FrpServer) error {
	authConfig := configv1.AuthClientConfig{
		Token:  obj.Spec.Auth.Token,
		Method: configv1.AuthMethod(obj.Spec.Auth.Method),
	}
	if obj.Spec.Auth.OIDC != nil {
		authConfig.OIDC = configv1.AuthOIDCClientConfig{
			ClientID:                 obj.Spec.Auth.OIDC.ClientID,
			ClientSecret:             obj.Spec.Auth.OIDC.ClientSecret,
			Audience:                 obj.Spec.Auth.OIDC.Audience,
			Scope:                    obj.Spec.Auth.OIDC.Scope,
			TokenEndpointURL:         obj.Spec.Auth.OIDC.TokenEndpointURL,
			AdditionalEndpointParams: obj.Spec.Auth.OIDC.AdditionalEndpointParams,
		}
	}
	for _, scope := range obj.Spec.Auth.AdditionalScopes {
		authConfig.AdditionalScopes = append(authConfig.AdditionalScopes, configv1.AuthScope(scope))
	}
	tlsOptions := configv1.TLSClientConfig{
		Enable: lo.ToPtr(obj.Spec.Transport.TLS.SecretRef != nil),
		TLSConfig: configv1.TLSConfig{
			ServerName: obj.Spec.Transport.TLS.ServerName,
		},
		DisableCustomTLSFirstByte: obj.Spec.Transport.TLS.DisableCustomTLSFirstByte,
	}
	transportConfig := configv1.ClientTransportConfig{
		TLS:                     tlsOptions,
		Protocol:                string(obj.Spec.Transport.Protocol),
		DialServerTimeout:       obj.Spec.Transport.DialServerTimeout,
		DialServerKeepAlive:     obj.Spec.Transport.DialServerKeepAlive,
		ConnectServerLocalIP:    obj.Spec.Transport.ConnectServerLocalIP,
		ProxyURL:                obj.Spec.Transport.ProxyURL,
		PoolCount:               obj.Spec.Transport.PoolCount,
		TCPMux:                  obj.Spec.Transport.TCPMux,
		TCPMuxKeepaliveInterval: obj.Spec.Transport.TCPMuxKeepaliveInterval,
		HeartbeatInterval:       obj.Spec.Transport.HeartbeatInterval,
		HeartbeatTimeout:        obj.Spec.Transport.HeartbeatTimeout,
	}
	if obj.Spec.Transport.QUIC != nil {
		transportConfig.QUIC = configv1.QUICOptions{
			KeepalivePeriod:    obj.Spec.Transport.QUIC.KeepalivePeriod,
			MaxIdleTimeout:     obj.Spec.Transport.QUIC.MaxIdleTimeout,
			MaxIncomingStreams: obj.Spec.Transport.QUIC.MaxIncomingStreams,
		}
	}
	commonConfig := configv1.ClientCommonConfig{
		User:              obj.Spec.User,
		Auth:              authConfig,
		Transport:         transportConfig,
		ServerAddr:        obj.Spec.ServerAddr,
		ServerPort:        obj.Spec.ServerPort,
		NatHoleSTUNServer: obj.Spec.NatHoleSTUNServer,
		DNSServer:         obj.Spec.DNSServer,
		LoginFailExit:     obj.Spec.LoginFailExit,
		UDPPacketSize:     obj.Spec.UDPPacketSize,
		Metadatas:         obj.Spec.Metadatas,
	}
	if obj.Spec.Transport.TLS.SecretRef != nil {
		secretObj := &v1.Secret{}
		secretObjKey := client.ObjectKey{
			Name:      obj.Spec.Transport.TLS.SecretRef.Name,
			Namespace: obj.Spec.Transport.TLS.SecretRef.Namespace,
		}
		commonConfig.Transport.TLS.Enable = lo.ToPtr(true)

		if err := cli.Get(ctx, secretObjKey, secretObj); err != nil {
			return fmt.Errorf("unable get secret '%+v', got: '%w'", secretObjKey, err)
		}

		certFile, err := os.CreateTemp(os.TempDir(), "cert")
		if err != nil {
			return fmt.Errorf("unable create temp file, got: '%w'", err)
		}
		defer func() {
			_ = certFile.Close()
			_ = os.Remove(certFile.Name())
		}()

		certData, ok := secretObj.Data[v1beta1.DefaultCertFileName]
		if !ok {
			return fmt.Errorf("file '%s' not found on secret '%+v', got: %w", v1beta1.DefaultCertFileName, secretObjKey, err)
		}

		_, err = certFile.Write(certData)
		if err != nil {
			return fmt.Errorf("file '%s' has incorrect content on secret '%+v', got: %w", v1beta1.DefaultCertFileName, secretObjKey, err)
		}

		commonConfig.Transport.TLS.CertFile = certFile.Name()

		keyFile, err := os.CreateTemp(os.TempDir(), "key")
		if err != nil {
			return fmt.Errorf("unable create temp file, got: '%w'", err)
		}
		defer func() {
			_ = keyFile.Close()
			_ = os.Remove(keyFile.Name())
		}()

		keyData, ok := secretObj.Data[v1beta1.DefaultKeyFileName]
		if !ok {
			return fmt.Errorf("file '%s' not found on secret '%+v', got: %w", v1beta1.DefaultKeyFileName, secretObjKey, err)
		}

		_, err = keyFile.Write(keyData)
		if err != nil {
			return fmt.Errorf("file '%s' has incorrect content on secret '%+v', got: %w", v1beta1.DefaultCertFileName, secretObjKey, err)
		}

		commonConfig.Transport.TLS.KeyFile = keyFile.Name()

		caData, ok := secretObj.Data[v1beta1.DefaultCaFileName]
		if ok {
			caFile, err := os.CreateTemp(os.TempDir(), "ca")
			if err != nil {
				return fmt.Errorf("unable create temp file, got: '%w'", err)
			}
			defer func() {
				_ = caFile.Close()
				_ = os.Remove(caFile.Name())
			}()

			_, err = caFile.Write(caData)
			if err != nil {
				return fmt.Errorf("file '%s' has incorrect content on secret '%+v', got: %w", v1beta1.DefaultCaFileName, secretObjKey, err)
			}
			commonConfig.Transport.TLS.TrustedCaFile = caFile.Name()
		}
	}

	commonConfig.Complete()

	_, err := validation.ValidateClientCommonConfig(&commonConfig)
	if err != nil {
		return err
	}
	var (
		loginRespMsg msg.LoginResp
		logger       = log.FromContext(ctx)
		authSetter   = auth.NewAuthSetter(commonConfig.Auth)
	)
	connMgr := frpclient.NewConnectionManager(ctx, &commonConfig)
	defer func() {
		_ = connMgr.Close()
	}()

	if err := connMgr.OpenConnection(); err != nil {
		logger.Error(err, "Error open frp connection manager conn")
		return err
	}

	conn, err := connMgr.Connect()
	if err != nil {
		logger.Error(err, "Unable create conn for connection manager")
		return err
	}
	defer func() {
		_ = conn.Close()
	}()

	hostname, err := os.Hostname()
	if err != nil {
		logger.Error(err, "Unable get hostname")
		return err
	}

	loginMsg := &msg.Login{
		Version:   version.Full(),
		Hostname:  hostname,
		Os:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		User:      commonConfig.User,
		Timestamp: time.Now().Unix(),
		PoolCount: commonConfig.Transport.PoolCount,
	}

	if err := authSetter.SetLogin(loginMsg); err != nil {
		logger.Error(err, "Error set login message")
		return err
	}

	if err = msg.WriteMsg(conn, loginMsg); err != nil {
		logger.Error(err, "Error write login message")
		return err
	}

	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	if err = msg.ReadMsgInto(conn, &loginRespMsg); err != nil {
		logger.Error(err, "Error to read login response")
		return err
	}
	_ = conn.SetReadDeadline(time.Time{})

	if loginRespMsg.Error != "" {
		logger.Error(err, "Error to login frp server")
		return fmt.Errorf(loginRespMsg.Error)
	}
	return nil
}
