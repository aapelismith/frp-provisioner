/*
Copyright 2023 Aapeli.Smith<aapeli.nian@gmail.com>.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FrpServerAuthMethod is the auth method for current FrpServer
// +enum
type FrpServerAuthMethod string

// FrpServerPhase is the status of a  FrpServer at the current time.
// +enum
type FrpServerPhase string

// FrpServerTransportProtocol specifies the protocol to use when interacting with the server.
// Valid values are "tcp", "kcp", "quic", "websocket" and "wss". By default, this value
// is "tcp".
type FrpServerTransportProtocol string

// FrpServerAuthScope is additional scope in auth info
// +enum
type FrpServerAuthScope string

const (
	// FrpServerAuthMethodToken means that the FRP server uses the token method to log in
	FrpServerAuthMethodToken FrpServerAuthMethod = "token"
	// FrpServerAuthMethodOIDC means that the FRP server uses the OIDC method to log in
	FrpServerAuthMethodOIDC FrpServerAuthMethod = "oidc"
)

const (
	FrpServerAuthScopeHeartBeats   FrpServerAuthScope = "HeartBeats"
	FrpServerAuthScopeNewWorkConns FrpServerAuthScope = "NewWorkConns"
)

const (
	FrpServerTransportProtocolTCP       FrpServerTransportProtocol = "tcp"
	FrpServerTransportProtocolKCP       FrpServerTransportProtocol = "kcp"
	FrpServerTransportProtocolQUIC      FrpServerTransportProtocol = "quic"
	FrpServerTransportProtocolWebsocket FrpServerTransportProtocol = "websocket"
	FrpServerTransportProtocolWSS       FrpServerTransportProtocol = "wss"
)

// These are the valid statuses of pods.
const (
	// FrpServerPending means the frp server object has been accepted by the system, but health testing has not started yet
	FrpServerPending FrpServerPhase = "Pending"
	// FrpServerHealthy means that the health check of the FRP server has passed
	FrpServerHealthy FrpServerPhase = "Healthy"
	// FrpServerUnhealthy Means that the health check of the FRP server has not been passed
	FrpServerUnhealthy FrpServerPhase = "Unhealthy"
	// FrpServerUnknown means that for some reason the state of the pod could not be obtained, typically due
	// to an error in communicating with the host of the FrpServer.
	FrpServerUnknown FrpServerPhase = "Unknown"
)

type FrpServerAuth struct {
	// Method specifies what authentication method to use to
	// authenticate frpc with frps. If "token" is specified - token will be
	// read into login message. If "oidc" is specified - OIDC (Open ID Connect)
	// token will be issued using OIDC settings. By default, this value is "token".
	Method FrpServerAuthMethod `json:"method,omitempty"`
	// AdditionalScopes specify whether to include auth info in additional scope.
	// Current supported scopes are: "HeartBeats", "NewWorkConns".
	// +optional
	AdditionalScopes []FrpServerAuthScope `json:"additionalScopes,omitempty"`
	// Token specifies the authorization token used to create keys to be sent
	// to the server. The server must have a matching token for authorization
	// to succeed.  By default, this value is "".
	Token string `json:"token,omitempty"`
	// +optional
	OIDC *FrpServerAuthOIDC `json:"oidc,omitempty"`
}

type FrpServerAuthOIDC struct {
	// ClientID specifies the client ID to use to get a token in OIDC authentication.
	ClientID string `json:"clientID,omitempty"`
	// ClientSecret specifies the client secret to use to get a token in OIDC
	// authentication.
	ClientSecret string `json:"clientSecret,omitempty"`
	// Audience specifies the audience of the token in OIDC authentication.
	Audience string `json:"audience,omitempty"`
	// Scope specifies the scope of the token in OIDC authentication.
	Scope string `json:"scope,omitempty"`
	// TokenEndpointURL specifies the URL which implements OIDC Token Endpoint.
	// It will be used to get an OIDC token.
	// +optional
	TokenEndpointURL string `json:"tokenEndpointURL,omitempty"`
	// AdditionalEndpointParams specifies additional parameters to be sent
	// this field will be transfer to map[string][]string in OIDC token generator.
	// +optional
	AdditionalEndpointParams map[string]string `json:"additionalEndpointParams,omitempty"`
}

type FrpServerTransport struct {
	// Protocol specifies the protocol to use when interacting with the server.
	// Valid values are "tcp", "kcp", "quic", "websocket" and "wss". By default, this value
	// is "tcp".
	Protocol FrpServerTransportProtocol `json:"protocol,omitempty"`
	// The maximum amount of time a dial to server will wait for a connect to complete.
	DialServerTimeout int64 `json:"dialServerTimeout,omitempty"`
	// DialServerKeepAlive specifies the interval between keep-alive probes for an active network connection between frpc and frps.
	// If negative, keep-alive probes are disabled.
	DialServerKeepAlive int64 `json:"dialServerKeepalive,omitempty"`
	// ConnectServerLocalIP specifies the address of the client bind when it connect to server.
	// Note: This value only use in TCP/Websocket protocol. Not support in KCP protocol.
	ConnectServerLocalIP string `json:"connectServerLocalIP,omitempty"`
	// ProxyURL specifies a proxy address to connect to the server through. If
	// this value is "", the server will be connected directly. By default,
	// this value is read from the "http_proxy" environment variable.
	ProxyURL string `json:"proxyURL,omitempty"`
	// PoolCount specifies the number of connections the client will make to
	// the server in advance.
	PoolCount int `json:"poolCount,omitempty"`
	// TCPMux toggles TCP stream multiplexing. This allows multiple requests
	// from a client to share a single TCP connection. If this value is true,
	// the server must have TCP multiplexing enabled as well. By default, this
	// value is true.
	TCPMux *bool `json:"tcpMux,omitempty"`
	// TCPMuxKeepaliveInterval specifies the keep alive interval for TCP stream multipler.
	// If TCPMux is true, heartbeat of application layer is unnecessary because it can only rely on heartbeat in TCPMux.
	TCPMuxKeepaliveInterval int64 `json:"tcpMuxKeepaliveInterval,omitempty"`
	// QUIC protocol options.
	QUIC *FrpServerTransportQUIC `json:"quic,omitempty"`
	// HeartBeatInterval specifies at what interval heartbeats are sent to the
	// server, in seconds. It is not recommended to change this value. By
	// default, this value is 30. Set negative value to disable it.
	HeartbeatInterval int64 `json:"heartbeatInterval,omitempty"`
	// HeartBeatTimeout specifies the maximum allowed heartbeat response delay
	// before the connection is terminated, in seconds. It is not recommended
	// to change this value. By default, this value is 90. Set negative value to disable it.
	HeartbeatTimeout int64 `json:"heartbeatTimeout,omitempty"`
	// TLS specifies TLS settings for the connection to the server.
	TLS FrpServerTransportTLS `json:"tls,omitempty"`
}

// FrpServerTransportQUIC the protocol options
type FrpServerTransportQUIC struct {
	KeepalivePeriod    int `json:"keepalivePeriod,omitempty"`
	MaxIdleTimeout     int `json:"maxIdleTimeout,omitempty"`
	MaxIncomingStreams int `json:"maxIncomingStreams,omitempty"`
}

type FrpServerTransportTLS struct {
	// Enable specifies whether TLS should be used when communicating
	// with the server. If "tls.certFile" and "tls.keyFile" are valid,
	// client will load the supplied tls configuration.
	// Since v0.50.0, the default value has been changed to true, and tls is enabled by default.
	Enable *bool `json:"enable,omitempty"`
	// SecretRef is name of the tls secret for transport. It provided tls key, cert and CA file
	SecretRef *v1.SecretReference `json:"secretRef,omitempty"`
	// CertFileName specifies the filename of the cert file on secretRef.
	CertFileName string `json:"certFileName,omitempty"`
	// KeyFileName specifies the filename of the secret key file on secretRef.
	KeyFileName string `json:"keyFileName,omitempty"`
	// CaFileName specifies the filename of the trusted ca file on secretRef.
	CaFileName string `json:"caFileName,omitempty"`
	// ServerName specifies the custom server name of tls certificate. By
	// default, server name if same to ServerAddr.
	ServerName string `json:"serverName,omitempty"`
	// If DisableCustomTLSFirstByte is set to false, frpc will establish a connection with frps using the
	// first custom byte when tls is enabled.
	// Since v0.50.0, the default value has been changed to true, and the first custom byte is disabled by default.
	DisableCustomTLSFirstByte *bool `json:"disableCustomTLSFirstByte,omitempty"`
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FrpServerSpec defines the desired state of FrpServer
type FrpServerSpec struct {
	// the auth config for current FrpServer
	Auth FrpServerAuth `json:"auth,omitempty"`
	// User specifies a prefix for proxy names to distinguish them from other
	// clients. If this value is not "", proxy names will automatically be
	// changed to "{user}.{proxy_name}".
	User string `json:"user,omitempty"`
	// ServerAddr specifies the address of the server to connect to. By
	// default, this value is "0.0.0.0".
	ServerAddr string `json:"serverAddr,omitempty"`
	// ServerPort specifies the port to connect to the server on. By default,
	// this value is 7000.
	ServerPort int `json:"serverPort,omitempty"`
	// ExternalIPs is set for load-balancer ingress points that are DNS/IP based
	ExternalIPs []string `json:"externalIPs,omitempty"`
	// STUN server to help penetrate NAT hole.
	NatHoleSTUNServer string `json:"natHoleStunServer,omitempty"`
	// DNSServer specifies a DNS server address for FRPC to use. If this value
	// is "", the default DNS will be used.
	DNSServer string `json:"dnsServer,omitempty"`
	// LoginFailExit controls whether the client should exit after a
	// failed login attempt. If false, the client will retry until a login
	// attempt succeeds. By default, this value is true.
	LoginFailExit *bool              `json:"loginFailExit,omitempty"`
	Transport     FrpServerTransport `json:"transport,omitempty"`
	// UDPPacketSize specifies the udp packet size
	// By default, this value is 1500
	UDPPacketSize int64 `json:"udpPacketSize,omitempty"`
	// Client metadata info
	Metadatas map[string]string `json:"metadatas,omitempty"`
}

// FrpServerStatus defines the observed state of FrpServer
type FrpServerStatus struct {
	// The phase of a FrpServer is a simple, high-level summary of where the FrpServer is in its lifecycle.
	Phase FrpServerPhase `json:"phase"`
	// Current service state
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
	// Reason A brief CamelCase message indicating details about why the pod is in this state.
	// +optional
	Reason string `json:"reason,omitempty"`
	// Services is a list of all services
	// +optional
	ServiceReferences []v1.TypedObjectReference `json:"serviceReferences,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// FrpServer is the Schema for the frpservers API
type FrpServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FrpServerSpec   `json:"spec,omitempty"`
	Status FrpServerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// FrpServerList contains a list of FrpServer
type FrpServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FrpServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FrpServer{}, &FrpServerList{})
}
