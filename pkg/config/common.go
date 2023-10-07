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

package config

import (
	"github.com/aapelismith/frp-provisioner/pkg/utils"
)

type AuthScope string

const (
	AuthScopeHeartBeats   AuthScope = "HeartBeats"
	AuthScopeNewWorkConns AuthScope = "NewWorkConns"
)

type AuthMethod string

const (
	AuthMethodToken AuthMethod = "token"
	AuthMethodOIDC  AuthMethod = "oidc"
)

var (
	SupportedAuthMethods          = []AuthMethod{AuthMethodToken, AuthMethodOIDC}
	SupportedAuthAdditionalScopes = []AuthScope{AuthScopeHeartBeats, AuthScopeNewWorkConns}
	SupportedTransportProtocols   = []string{"tcp", "kcp", "quic", "websocket", "wss"}
)

// QUICOptions  QUIC protocol options
type QUICOptions struct {
	KeepalivePeriod    int `json:"keepalivePeriod,omitempty"`
	MaxIdleTimeout     int `json:"maxIdleTimeout,omitempty"`
	MaxIncomingStreams int `json:"maxIncomingStreams,omitempty"`
}

func (c *QUICOptions) SetDefaults() {
	c.KeepalivePeriod = utils.EmptyOr(c.KeepalivePeriod, 10)
	c.MaxIdleTimeout = utils.EmptyOr(c.MaxIdleTimeout, 30)
	c.MaxIncomingStreams = utils.EmptyOr(c.MaxIncomingStreams, 100000)
}

type TLSConfig struct {
	// CertPath specifies the path of the cert file that client will load.
	CertFile string `json:"certFile,omitempty"`
	// KeyPath specifies the path of the secret key file that client will load.
	KeyFile string `json:"keyFile,omitempty"`
	// TrustedCaFile specifies the path of the trusted ca file that will load.
	TrustedCaFile string `json:"trustedCaFile,omitempty"`
	// ServerName specifies the custom server name of tls certificate. By
	// default, server name if same to ServerAddr.
	ServerName string `json:"serverName,omitempty"`
}

type HTTPPluginOptions struct {
	Name      string   `json:"name"`
	Addr      string   `json:"addr"`
	Path      string   `json:"path"`
	Ops       []string `json:"ops"`
	TLSVerify bool     `json:"tlsVerify,omitempty"`
}

type HeaderOperations struct {
	Set map[string]string `json:"set,omitempty"`
}
