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
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/frp-sigs/frp-provisioner/pkg/utils"
	"reflect"

	"github.com/samber/lo"

	"github.com/fatedier/frp/pkg/config/types"
)

type ProxyTransport struct {
	// UseEncryption controls whether communication with the server will
	// be encrypted. Encryption is done using the tokens supplied in the server
	// and client configuration.
	UseEncryption bool `json:"useEncryption,omitempty"`
	// UseCompression controls whether communication with the server
	// will be compressed.
	UseCompression bool `json:"useCompression,omitempty"`
	// BandwidthLimit limit the bandwidth
	// 0 means no limit
	BandwidthLimit types.BandwidthQuantity `json:"bandwidthLimit,omitempty"`
	// BandwidthLimitMode specifies whether to limit the bandwidth on the
	// client or server side. Valid values include "client" and "server".
	// By default, this value is "client".
	BandwidthLimitMode string `json:"bandwidthLimitMode,omitempty"`
	// ProxyProtocolVersion specifies which protocol version to use. Valid
	// values include "msg", "v2", and "". If the value is "", a protocol
	// version will be automatically selected. By default, this value is "".
	ProxyProtocolVersion string `json:"proxyProtocolVersion,omitempty"`
}

type LoadBalancerConfig struct {
	// Group specifies which group the is a part of. The server will use
	// this information to load balance proxies in the same group. If the value
	// is "", this will not be in a group.
	Group string `json:"group"`
	// GroupKey specifies a group key, which should be the same among proxies
	// of the same group.
	GroupKey string `json:"groupKey,omitempty"`
}

type ProxyBackend struct {
	// LocalIP specifies the IP address or host name of the backend.
	LocalIP string `json:"localIP,omitempty"`
	// LocalPort specifies the port of the backend.
	LocalPort int `json:"localPort,omitempty"`

	// Plugin specifies what plugin should be used for handling connections. If this value
	// is set, the LocalIP and LocalPort values will be ignored.
	Plugin TypedClientPluginOptions `json:"plugin,omitempty"`
}

// HealthCheckConfig configures health checking. This can be useful for load
// balancing purposes to detect and remove proxies to failing services.
type HealthCheckConfig struct {
	// Type specifies what protocol to use for health checking.
	// Valid values include "tcp", "http", and "". If this value is "", health
	// checking will not be performed.
	//
	// If the type is "tcp", a connection will be attempted to the target
	// server. If a connection cannot be established, the health check fails.
	//
	// If the type is "http", a GET request will be made to the endpoint
	// specified by HealthCheckURL. If the response is not a 200, the health
	// check fails.
	Type string `json:"type"` // tcp | http
	// TimeoutSeconds specifies the number of seconds to wait for a health
	// check attempt to connect. If the timeout is reached, this counts as a
	// health check failure. By default, this value is 3.
	TimeoutSeconds int `json:"timeoutSeconds,omitempty"`
	// MaxFailed specifies the number of allowed failures before the
	// is stopped. By default, this value is 1.
	MaxFailed int `json:"maxFailed,omitempty"`
	// IntervalSeconds specifies the time in seconds between health
	// checks. By default, this value is 10.
	IntervalSeconds int `json:"intervalSeconds"`
	// Path specifies the path to send health checks to if the
	// health check type is "http".
	Path string `json:"path,omitempty"`
}

type DomainConfig struct {
	CustomDomains []string `json:"customDomains,omitempty"`
	SubDomain     string   `json:"subdomain,omitempty"`
}

func (c *DomainConfig) Validate() error {
	if c.SubDomain == "" && len(c.CustomDomains) == 0 {
		return errors.New("subdomain and custom domains should not be both empty")
	}
	return nil
}

type ProxyBaseConfig struct {
	Name      string         `json:"name"`
	Type      string         `json:"type"`
	Transport ProxyTransport `json:"transport,omitempty"`
	// metadata info for each proxy
	Metadatas    map[string]string  `json:"metadatas,omitempty"`
	LoadBalancer LoadBalancerConfig `json:"loadBalancer,omitempty"`
	HealthCheck  HealthCheckConfig  `json:"healthCheck,omitempty"`
	ProxyBackend
}

func (c *ProxyBaseConfig) GetBaseConfig() *ProxyBaseConfig {
	return c
}

func (c *ProxyBaseConfig) SetDefaults(namePrefix string) {
	c.Name = lo.Ternary(namePrefix == "", "", namePrefix+".") + c.Name
	c.LocalIP = utils.EmptyOr(c.LocalIP, "127.0.0.1")
	c.Transport.BandwidthLimitMode = utils.EmptyOr(c.Transport.BandwidthLimitMode, types.BandwidthLimitModeClient)
}

func (c *ProxyBaseConfig) MarshalToMsg(m *msg.NewProxy) {
	m.ProxyName = c.Name
	m.ProxyType = c.Type
	m.UseEncryption = c.Transport.UseEncryption
	m.UseCompression = c.Transport.UseCompression
	m.BandwidthLimit = c.Transport.BandwidthLimit.String()
	// leave it empty for default value to reduce traffic
	if c.Transport.BandwidthLimitMode != "client" {
		m.BandwidthLimitMode = c.Transport.BandwidthLimitMode
	}
	m.Group = c.LoadBalancer.Group
	m.GroupKey = c.LoadBalancer.GroupKey
	m.Metas = c.Metadatas
}

func (c *ProxyBaseConfig) UnmarshalFromMsg(m *msg.NewProxy) {
	c.Name = m.ProxyName
	c.Type = m.ProxyType
	c.Transport.UseEncryption = m.UseEncryption
	c.Transport.UseCompression = m.UseCompression
	if m.BandwidthLimit != "" {
		c.Transport.BandwidthLimit, _ = types.NewBandwidthQuantity(m.BandwidthLimit)
	}
	if m.BandwidthLimitMode != "" {
		c.Transport.BandwidthLimitMode = m.BandwidthLimitMode
	}
	c.LoadBalancer.Group = m.Group
	c.LoadBalancer.GroupKey = m.GroupKey
	c.Metadatas = m.Metas
}

func (c *ProxyBaseConfig) Validate() error {
	if c.Name == "" {
		return errors.New("name should not be empty")
	}

	if !lo.Contains([]string{"", "msg", "v2"}, c.Transport.ProxyProtocolVersion) {
		return fmt.Errorf("not support proxy protocol version: %s", c.Transport.ProxyProtocolVersion)
	}

	if !lo.Contains([]string{"client", "server"}, c.Transport.BandwidthLimitMode) {
		return fmt.Errorf("bandwidth limit mode should be client or server")
	}

	if c.Plugin.Type == "" {
		if err := utils.ValidatePort(c.LocalPort, "localPort"); err != nil {
			return fmt.Errorf("localPort: %v", err)
		}
	}

	if !lo.Contains([]string{"", "tcp", "http"}, c.HealthCheck.Type) {
		return fmt.Errorf("not support health check type: %s", c.HealthCheck.Type)
	}
	if c.HealthCheck.Type != "" {
		if c.HealthCheck.Type == "http" &&
			c.HealthCheck.Path == "" {
			return fmt.Errorf("health check path should not be empty")
		}
	}

	if c.Plugin.Type != "" {
		if err := c.Plugin.ClientPluginOptions.Validate(); err != nil {
			return fmt.Errorf("plugin %s: %v", c.Plugin.Type, err)
		}
	}
	return nil
}

type TypedProxyConfig struct {
	Type string `json:"type"`
	ProxyConfigurer
}

func (c *TypedProxyConfig) UnmarshalJSON(b []byte) error {
	if len(b) == 4 && string(b) == "null" {
		return errors.New("type is required")
	}

	typeStruct := struct {
		Type string `json:"type"`
	}{}
	if err := json.Unmarshal(b, &typeStruct); err != nil {
		return err
	}

	c.Type = typeStruct.Type
	configurer := NewProxyConfigurerByType(ProxyType(typeStruct.Type))
	if configurer == nil {
		return fmt.Errorf("unknown proxy type: %s", typeStruct.Type)
	}
	if err := json.Unmarshal(b, configurer); err != nil {
		return err
	}
	c.ProxyConfigurer = configurer
	return nil
}

type ProxyConfigurer interface {
	// Validate and check if the configuration is correct
	Validate() error
	// SetDefaults set default values for current config.
	SetDefaults(namePrefix string)
	// GetBaseConfig get base config from ProxyConfigurer
	GetBaseConfig() *ProxyBaseConfig
	// MarshalToMsg marshals this config into a api.NewProxy message. This
	// function will be called on the frpc side.
	MarshalToMsg(*msg.NewProxy)
	// UnmarshalFromMsg unmarshal a api.NewProxy message into this config.
	// This function will be called on the frps side.
	UnmarshalFromMsg(*msg.NewProxy)
}

type ProxyType string

const (
	ProxyTypeTCP    ProxyType = "tcp"
	ProxyTypeUDP    ProxyType = "udp"
	ProxyTypeTCPMUX ProxyType = "tcpmux"
	ProxyTypeHTTP   ProxyType = "http"
	ProxyTypeHTTPS  ProxyType = "https"
	ProxyTypeSTCP   ProxyType = "stcp"
	ProxyTypeXTCP   ProxyType = "xtcp"
	ProxyTypeSUDP   ProxyType = "sudp"
)

var proxyConfigTypeMap = map[ProxyType]reflect.Type{
	ProxyTypeTCP:    reflect.TypeOf(TCPProxyConfig{}),
	ProxyTypeUDP:    reflect.TypeOf(UDPProxyConfig{}),
	ProxyTypeHTTP:   reflect.TypeOf(HTTPProxyConfig{}),
	ProxyTypeHTTPS:  reflect.TypeOf(HTTPSProxyConfig{}),
	ProxyTypeTCPMUX: reflect.TypeOf(TCPMuxProxyConfig{}),
	ProxyTypeSTCP:   reflect.TypeOf(STCPProxyConfig{}),
	ProxyTypeXTCP:   reflect.TypeOf(XTCPProxyConfig{}),
	ProxyTypeSUDP:   reflect.TypeOf(SUDPProxyConfig{}),
}

func NewProxyConfigurerByType(proxyType ProxyType) ProxyConfigurer {
	v, ok := proxyConfigTypeMap[proxyType]
	if !ok {
		return nil
	}
	return reflect.New(v).Interface().(ProxyConfigurer)
}

var _ ProxyConfigurer = &TCPProxyConfig{}

type TCPProxyConfig struct {
	ProxyBaseConfig

	RemotePort int `json:"remotePort,omitempty"`
}

func (c *TCPProxyConfig) Validate() error {
	return c.ProxyBaseConfig.Validate()
}

func (c *TCPProxyConfig) MarshalToMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.MarshalToMsg(m)

	m.RemotePort = c.RemotePort
}

func (c *TCPProxyConfig) UnmarshalFromMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.UnmarshalFromMsg(m)

	c.RemotePort = m.RemotePort
}

var _ ProxyConfigurer = &UDPProxyConfig{}

type UDPProxyConfig struct {
	ProxyBaseConfig

	RemotePort int `json:"remotePort,omitempty"`
}

func (c *UDPProxyConfig) Validate() error {
	return c.ProxyBaseConfig.Validate()
}

func (c *UDPProxyConfig) MarshalToMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.MarshalToMsg(m)

	m.RemotePort = c.RemotePort
}

func (c *UDPProxyConfig) UnmarshalFromMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.UnmarshalFromMsg(m)

	c.RemotePort = m.RemotePort
}

var _ ProxyConfigurer = &HTTPProxyConfig{}

type HTTPProxyConfig struct {
	ProxyBaseConfig
	DomainConfig

	Locations         []string         `json:"locations,omitempty"`
	HTTPUser          string           `json:"httpUser,omitempty"`
	HTTPPassword      string           `json:"httpPassword,omitempty"`
	HostHeaderRewrite string           `json:"hostHeaderRewrite,omitempty"`
	RequestHeaders    HeaderOperations `json:"requestHeaders,omitempty"`
	RouteByHTTPUser   string           `json:"routeByHTTPUser,omitempty"`
}

func (c *HTTPProxyConfig) MarshalToMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.MarshalToMsg(m)

	m.CustomDomains = c.CustomDomains
	m.SubDomain = c.SubDomain
	m.Locations = c.Locations
	m.HostHeaderRewrite = c.HostHeaderRewrite
	m.HTTPUser = c.HTTPUser
	m.HTTPPwd = c.HTTPPassword
	m.Headers = c.RequestHeaders.Set
	m.RouteByHTTPUser = c.RouteByHTTPUser
}

func (c *HTTPProxyConfig) UnmarshalFromMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.UnmarshalFromMsg(m)

	c.CustomDomains = m.CustomDomains
	c.SubDomain = m.SubDomain
	c.Locations = m.Locations
	c.HostHeaderRewrite = m.HostHeaderRewrite
	c.HTTPUser = m.HTTPUser
	c.HTTPPassword = m.HTTPPwd
	c.RequestHeaders.Set = m.Headers
	c.RouteByHTTPUser = m.RouteByHTTPUser
}

func (c *HTTPProxyConfig) Validate() error {
	return c.ProxyBaseConfig.Validate()
}

var _ ProxyConfigurer = &HTTPSProxyConfig{}

type HTTPSProxyConfig struct {
	ProxyBaseConfig
	DomainConfig
}

func (c *HTTPSProxyConfig) MarshalToMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.MarshalToMsg(m)

	m.CustomDomains = c.CustomDomains
	m.SubDomain = c.SubDomain
}

func (c *HTTPSProxyConfig) UnmarshalFromMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.UnmarshalFromMsg(m)

	c.CustomDomains = m.CustomDomains
	c.SubDomain = m.SubDomain
}

func (c *HTTPSProxyConfig) Validate() error {
	return c.ProxyBaseConfig.Validate()
}

type TCPMultiplexerType string

const (
	TCPMultiplexerHTTPConnect TCPMultiplexerType = "httpconnect"
)

var _ ProxyConfigurer = &TCPMuxProxyConfig{}

type TCPMuxProxyConfig struct {
	ProxyBaseConfig
	DomainConfig

	HTTPUser        string `json:"httpUser,omitempty"`
	HTTPPassword    string `json:"httpPassword,omitempty"`
	RouteByHTTPUser string `json:"routeByHTTPUser,omitempty"`
	Multiplexer     string `json:"multiplexer,omitempty"`
}

func (c *TCPMuxProxyConfig) Validate() error {
	if err := c.DomainConfig.Validate(); err != nil {
		return err
	}

	if !lo.Contains([]string{string(TCPMultiplexerHTTPConnect)}, c.Multiplexer) {
		return fmt.Errorf("not support multiplexer: %s", c.Multiplexer)
	}
	return c.ProxyBaseConfig.Validate()
}

func (c *TCPMuxProxyConfig) MarshalToMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.MarshalToMsg(m)

	m.CustomDomains = c.CustomDomains
	m.SubDomain = c.SubDomain
	m.Multiplexer = c.Multiplexer
	m.HTTPUser = c.HTTPUser
	m.HTTPPwd = c.HTTPPassword
	m.RouteByHTTPUser = c.RouteByHTTPUser
}

func (c *TCPMuxProxyConfig) UnmarshalFromMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.UnmarshalFromMsg(m)

	c.CustomDomains = m.CustomDomains
	c.SubDomain = m.SubDomain
	c.Multiplexer = m.Multiplexer
	c.HTTPUser = m.HTTPUser
	c.HTTPPassword = m.HTTPPwd
	c.RouteByHTTPUser = m.RouteByHTTPUser
}

var _ ProxyConfigurer = &STCPProxyConfig{}

type STCPProxyConfig struct {
	ProxyBaseConfig

	SecretKey  string   `json:"secretKey,omitempty"`
	AllowUsers []string `json:"allowUsers,omitempty"`
}

func (c *STCPProxyConfig) Validate() error {
	return c.ProxyBaseConfig.Validate()
}

func (c *STCPProxyConfig) MarshalToMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.MarshalToMsg(m)

	m.Sk = c.SecretKey
	m.AllowUsers = c.AllowUsers
}

func (c *STCPProxyConfig) UnmarshalFromMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.UnmarshalFromMsg(m)

	c.SecretKey = m.Sk
	c.AllowUsers = m.AllowUsers
}

var _ ProxyConfigurer = &XTCPProxyConfig{}

type XTCPProxyConfig struct {
	ProxyBaseConfig

	SecretKey  string   `json:"secretKey,omitempty"`
	AllowUsers []string `json:"allowUsers,omitempty"`
}

func (c *XTCPProxyConfig) Validate() error {
	return c.ProxyBaseConfig.Validate()
}

func (c *XTCPProxyConfig) MarshalToMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.MarshalToMsg(m)

	m.Sk = c.SecretKey
	m.AllowUsers = c.AllowUsers
}

func (c *XTCPProxyConfig) UnmarshalFromMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.UnmarshalFromMsg(m)

	c.SecretKey = m.Sk
	c.AllowUsers = m.AllowUsers
}

var _ ProxyConfigurer = &SUDPProxyConfig{}

type SUDPProxyConfig struct {
	ProxyBaseConfig

	SecretKey  string   `json:"secretKey,omitempty"`
	AllowUsers []string `json:"allowUsers,omitempty"`
}

func (c *SUDPProxyConfig) Validate() error {
	return c.ProxyBaseConfig.Validate()
}

func (c *SUDPProxyConfig) MarshalToMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.MarshalToMsg(m)

	m.Sk = c.SecretKey
	m.AllowUsers = c.AllowUsers
}

func (c *SUDPProxyConfig) UnmarshalFromMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.UnmarshalFromMsg(m)

	c.SecretKey = m.Sk
	c.AllowUsers = m.AllowUsers
}
