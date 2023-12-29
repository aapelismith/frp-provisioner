// Copyright 2023 The frp Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package frp

import (
	"context"
	"fmt"
	configv1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/frp-sigs/frp-provisioner/pkg/api/v1beta1"
	"github.com/samber/lo"
	v1 "k8s.io/api/core/v1"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GenClientCommonConfig use v1beta1.FrpServer to generate v1.ClientCommonConfig, Please delete the file
// spec.transport.tls.caFileName/spec.transport.tls.certFileName/spec.transport.tls.keyFileName after use
func GenClientCommonConfig(ctx context.Context, cli client.Client, obj *v1beta1.FrpServer) (*configv1.ClientCommonConfig, error) {
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
		Enable: obj.Spec.Transport.TLS.Enable,
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
	if lo.FromPtr(obj.Spec.Transport.TLS.Enable) && obj.Spec.Transport.TLS.SecretRef != nil {
		secretObj := &v1.Secret{}
		secretObjKey := client.ObjectKey{
			Name:      obj.Spec.Transport.TLS.SecretRef.Name,
			Namespace: obj.Spec.Transport.TLS.SecretRef.Namespace,
		}

		if err := cli.Get(ctx, secretObjKey, secretObj); err != nil {
			return nil, fmt.Errorf("unable get secret '%+v', got: '%w'", secretObjKey, err)
		}

		certFile, err := os.CreateTemp(os.TempDir(), "cert")
		if err != nil {
			return nil, fmt.Errorf("unable create temp file, got: '%w'", err)
		}
		defer func() {
			_ = certFile.Close()
		}()

		certData, ok := secretObj.Data[obj.Spec.Transport.TLS.CertFileName]
		if !ok {
			return nil, fmt.Errorf("file '%s' not found on secret '%+v', got: %w", obj.Spec.Transport.TLS.CertFileName, secretObjKey, err)
		}

		_, err = certFile.Write(certData)
		if err != nil {
			return nil, fmt.Errorf("file '%s' has incorrect content on secret '%+v', got: %w", obj.Spec.Transport.TLS.CertFileName, secretObjKey, err)
		}

		commonConfig.Transport.TLS.CertFile = certFile.Name()

		keyFile, err := os.CreateTemp(os.TempDir(), "key")
		if err != nil {
			return nil, fmt.Errorf("unable create temp file, got: '%w'", err)
		}
		defer func() {
			_ = keyFile.Close()
		}()

		keyData, ok := secretObj.Data[obj.Spec.Transport.TLS.KeyFileName]
		if !ok {
			return nil, fmt.Errorf("file '%s' not found on secret '%+v', got: %w", obj.Spec.Transport.TLS.KeyFileName, secretObjKey, err)
		}

		_, err = keyFile.Write(keyData)
		if err != nil {
			return nil, fmt.Errorf("file '%s' has incorrect content on secret '%+v', got: %w", obj.Spec.Transport.TLS.KeyFileName, secretObjKey, err)
		}

		commonConfig.Transport.TLS.KeyFile = keyFile.Name()

		caFile, err := os.CreateTemp(os.TempDir(), "ca")
		if err != nil {
			return nil, fmt.Errorf("unable create temp file, got: '%w'", err)
		}
		defer func() {
			_ = caFile.Close()
		}()

		caData, ok := secretObj.Data[obj.Spec.Transport.TLS.CaFileName]
		if !ok {
			return nil, fmt.Errorf("file '%s' not found on secret '%+v'", obj.Spec.Transport.TLS.CaFileName, secretObjKey)
		}

		_, err = caFile.Write(caData)
		if err != nil {
			return nil, fmt.Errorf("file '%s' has incorrect content on secret '%+v', got: %w", obj.Spec.Transport.TLS.CaFileName, secretObjKey, err)
		}

		commonConfig.Transport.TLS.TrustedCaFile = caFile.Name()
	}
	commonConfig.Complete()
	return &commonConfig, nil
}
