package frp

import (
	"fmt"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/errors"
)

const defaultWorkspace = "/tmp/frp-service-provider"

type CommonConfig struct {
	// OidcClientID specifies the client ID to use to get a token in OIDC
	// authentication if AuthenticationMethod == "oidc". By default, this value
	// is "".
	OidcClientID string `ini:"oidc_client_id" json:"oidc_client_id"`
	// OidcClientSecret specifies the client secret to use to get a token in OIDC
	// authentication if AuthenticationMethod == "oidc". By default, this value
	// is "".
	OidcClientSecret string `ini:"oidc_client_secret" json:"oidc_client_secret"`
	// OidcAudience specifies the audience of the token in OIDC authentication
	// if AuthenticationMethod == "oidc". By default, this value is "".
	OidcAudience string `ini:"oidc_audience" json:"oidc_audience"`
	// OidcScope specifies the scope of the token in OIDC authentication
	// if AuthenticationMethod == "oidc". By default, this value is "".
	OidcScope string `ini:"oidc_scope" json:"oidc_scope"`
	// OidcTokenEndpointURL specifies the URL which implements OIDC Token Endpoint.
	// It will be used to get an OIDC token if AuthenticationMethod == "oidc".
	// By default, this value is "".
	OidcTokenEndpointURL string `ini:"oidc_token_endpoint_url" json:"oidc_token_endpoint_url"`

	// OidcAdditionalEndpointParams specifies additional parameters to be sent
	// this field will be transfer to map[string][]string in OIDC token generator
	// The field will be set by prefix "oidc_additional_"
	OidcAdditionalEndpointParams map[string]string `ini:"-" json:"oidc_additional_endpoint_params"`

	// AuthenticationMethod specifies what authentication method to use to
	// authenticate frpc with frps. If "token" is specified - token will be
	// read into login message. If "oidc" is specified - OIDC (Open ID Connect)
	// token will be issued using OIDC settings. By default, this value is "token".
	AuthenticationMethod string `ini:"authentication_method" json:"authentication_method"`
	// AuthenticateHeartBeats specifies whether to include authentication token in
	// heartbeats sent to frps. By default, this value is false.
	AuthenticateHeartBeats bool `ini:"authenticate_heartbeats" json:"authenticate_heartbeats"`
	// AuthenticateNewWorkConns specifies whether to include authentication token in
	// new work connections sent to frps. By default, this value is false.
	AuthenticateNewWorkConns bool `ini:"authenticate_new_work_conns" json:"authenticate_new_work_conns"`

	// Token specifies the authorization token used to create keys to be sent
	// to the server. The server must have a matching token for authorization
	// to succeed.  By default, this value is "".
	Token string `ini:"token" json:"token"`

	// ServerAddr specifies the address of the server to connect to. By
	// default, this value is "0.0.0.0".
	ServerAddr string `ini:"server_addr" json:"server_addr"`
	// ServerPort specifies the port to connect to the server on. By default,
	// this value is 7000.
	ServerPort int `ini:"server_port" json:"server_port"`
	// STUN server to help penetrate NAT hole.
	NatHoleSTUNServer string `ini:"nat_hole_stun_server" json:"nat_hole_stun_server"`
	// The maximum amount of time a dial to server will wait for a connect to complete.
	DialServerTimeout int64 `ini:"dial_server_timeout" json:"dial_server_timeout"`
	// DialServerKeepAlive specifies the interval between keep-alive probes for an active network connection between frpc and frps.
	// If negative, keep-alive probes are disabled.
	DialServerKeepAlive int64 `ini:"dial_server_keepalive" json:"dial_server_keepalive"`
	// ConnectServerLocalIP specifies the address of the client bind when it connect to server.
	// By default, this value is empty.
	// this value only use in TCP/Websocket protocol. Not support in KCP protocol.
	ConnectServerLocalIP string `ini:"connect_server_local_ip" json:"connect_server_local_ip"`
	// HTTPProxy specifies a proxy address to connect to the server through. If
	// this value is "", the server will be connected to directly. By default,
	// this value is read from the "http_proxy" environment variable.
	HTTPProxy string `ini:"http_proxy" json:"http_proxy"`
	// LogFile specifies a file where logs will be written to. This value will
	// only be used if LogWay is set appropriately. By default, this value is
	// "console".
	LogFile string `ini:"log_file" json:"log_file"`
	// LogWay specifies the way logging is managed. Valid values are "console"
	// or "file". If "console" is used, logs will be printed to stdout. If
	// "file" is used, logs will be printed to LogFile. By default, this value
	// is "console".
	LogWay string `ini:"log_way" json:"log_way"`
	// LogLevel specifies the minimum log level. Valid values are "trace",
	// "debug", "info", "warn", and "error". By default, this value is "info".
	LogLevel string `ini:"log_level" json:"log_level"`
	// LogMaxDays specifies the maximum number of days to store log information
	// before deletion. This is only used if LogWay == "file". By default, this
	// value is 0.
	LogMaxDays int64 `ini:"log_max_days" json:"log_max_days"`
	// DisableLogColor disables log colors when LogWay == "console" when set to
	// true. By default, this value is false.
	DisableLogColor bool `ini:"disable_log_color" json:"disable_log_color"`
	// AdminAddr specifies the address that the admin server binds to. By
	// default, this value is "127.0.0.1".
	AdminAddr string `ini:"admin_addr" json:"admin_addr"`
	// AdminPort specifies the port for the admin server to listen on. If this
	// value is 0, the admin server will not be started. By default, this value
	// is 0.
	AdminPort int `ini:"admin_port" json:"admin_port"`
	// AdminUser specifies the username that the admin server will use for
	// login.
	AdminUser string `ini:"admin_user" json:"admin_user"`
	// AdminPwd specifies the password that the admin server will use for
	// login.
	AdminPwd string `ini:"admin_pwd" json:"admin_pwd"`
	// AssetsDir specifies the local directory that the admin server will load
	// resources from. If this value is "", assets will be loaded from the
	// bundled executable using statik. By default, this value is "".
	AssetsDir string `ini:"assets_dir" json:"assets_dir"`
	// PoolCount specifies the number of connections the client will make to
	// the server in advance. By default, this value is 0.
	PoolCount int `ini:"pool_count" json:"pool_count"`
	// TCPMux toggles TCP stream multiplexing. This allows multiple requests
	// from a client to share a single TCP connection. If this value is true,
	// the server must have TCP multiplexing enabled as well. By default, this
	// value is true.
	TCPMux bool `ini:"tcp_mux" json:"tcp_mux"`
	// TCPMuxKeepaliveInterval specifies the keep alive interval for TCP stream multipler.
	// If TCPMux is true, heartbeat of application layer is unnecessary because it can only rely on heartbeat in TCPMux.
	TCPMuxKeepaliveInterval int64 `ini:"tcp_mux_keepalive_interval" json:"tcp_mux_keepalive_interval"`
	// User specifies a prefix for proxy names to distinguish them from other
	// clients. If this value is not "", proxy names will automatically be
	// changed to "{user}.{proxy_name}". By default, this value is "".
	User string `ini:"user" json:"user"`
	// DNSServer specifies a DNS server address for FRPC to use. If this value
	// is "", the default DNS will be used. By default, this value is "".
	DNSServer string `ini:"dns_server" json:"dns_server"`
	// LoginFailExit controls whether or not the client should exit after a
	// failed login attempt. If false, the client will retry until a login
	// attempt succeeds. By default, this value is true.
	LoginFailExit bool `ini:"login_fail_exit" json:"login_fail_exit"`
	// Start specifies a set of enabled proxies by name. If this set is empty,
	// all supplied proxies are enabled. By default, this value is an empty
	// set.
	Start []string `ini:"start" json:"start"`
	// Start map[string]struct{} `json:"start"`
	// Protocol specifies the protocol to use when interacting with the server.
	// Valid values are "tcp", "kcp", "quic", "websocket" and "wss". By default, this value
	// is "tcp".
	Protocol string `ini:"protocol" json:"protocol"`
	// QUIC protocol options
	QUICKeepalivePeriod    int `ini:"quic_keepalive_period" json:"quic_keepalive_period" validate:"gte=0"`
	QUICMaxIdleTimeout     int `ini:"quic_max_idle_timeout" json:"quic_max_idle_timeout" validate:"gte=0"`
	QUICMaxIncomingStreams int `ini:"quic_max_incoming_streams" json:"quic_max_incoming_streams" validate:"gte=0"`
	// TLSEnable specifies whether or not TLS should be used when communicating
	// with the server. If "tls_cert_file" and "tls_key_file" are valid,
	// client will load the supplied tls configuration.
	// Since v0.50.0, the default value has been changed to true, and tls is enabled by default.
	TLSEnable bool `ini:"tls_enable" json:"tls_enable"`
	// TLSCertPath specifies the path of the cert file that client will
	// load. It only works when "tls_enable" is true and "tls_key_file" is valid.
	TLSCertFile string `ini:"tls_cert_file" json:"tls_cert_file"`
	// TLSKeyPath specifies the path of the secret key file that client
	// will load. It only works when "tls_enable" is true and "tls_cert_file"
	// are valid.
	TLSKeyFile string `ini:"tls_key_file" json:"tls_key_file"`
	// TLSTrustedCaFile specifies the path of the trusted ca file that will load.
	// It only works when "tls_enable" is valid and tls configuration of server
	// has been specified.
	TLSTrustedCaFile string `ini:"tls_trusted_ca_file" json:"tls_trusted_ca_file"`
	// TLSServerName specifies the custom server name of tls certificate. By
	// default, server name if same to ServerAddr.
	TLSServerName string `ini:"tls_server_name" json:"tls_server_name"`
	// If the disable_custom_tls_first_byte is set to false, frpc will establish a connection with frps using the
	// first custom byte when tls is enabled.
	// Since v0.50.0, the default value has been changed to true, and the first custom byte is disabled by default.
	DisableCustomTLSFirstByte bool `ini:"disable_custom_tls_first_byte" json:"disable_custom_tls_first_byte"`
	// HeartBeatInterval specifies at what interval heartbeats are sent to the
	// server, in seconds. It is not recommended to change this value. By
	// default, this value is 30. Set negative value to disable it.
	HeartbeatInterval int64 `ini:"heartbeat_interval" json:"heartbeat_interval"`
	// HeartBeatTimeout specifies the maximum allowed heartbeat response delay
	// before the connection is terminated, in seconds. It is not recommended
	// to change this value. By default, this value is 90. Set negative value to disable it.
	HeartbeatTimeout int64 `ini:"heartbeat_timeout" json:"heartbeat_timeout"`
	// Client meta info
	Metas map[string]string `ini:"-" json:"metas" yaml:"metas"`
	// UDPPacketSize specifies the udp packet size
	// By default, this value is 1500
	UDPPacketSize int64 `ini:"udp_packet_size" json:"udp_packet_size" yaml:"udp_packet_size"`
	// Include other config files for proxies.
	IncludeConfigFiles []string `ini:"includes" json:"includes" yaml:"include_config_files"`
	// Enable golang pprof handlers in admin listener.
	// Admin port must be set first.
	PprofEnable bool `ini:"pprof_enable" json:"pprof_enable" yaml:"pprof_enable"`
}

// ServerConfig is the config for frp server.
type ServerConfig struct {
	// ServerName is the name of server.
	ServerName string `json:"server_name"`
}

// Options is the options for frp client.
type Options struct {
	// Workspace is the workspace for frpc.
	Workspace string `yaml:"workspace" json:"workspace"`
	// Servers is the config list for frpc common config.
	Servers []ServerConfig `yaml:"servers" json:"servers"`
}

// Validate validates the frp client options.
func (o *Options) Validate() error {
	errs := make([]error, 0)
	if o.Workspace == "" {
		errs = append(errs, fmt.Errorf("workspace cannot be empty"))
	}
	if len(o.Servers) == 0 {
		errs = append(errs, fmt.Errorf("servers cannot be empty"))
	}
	for index, srv := range o.Servers {
		if srv.ServerName == "" {
			errs = append(errs, fmt.Errorf("servers[%d].server_name cannot be empty", index))
		}
		if err := srv.ClientCommonConf.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("validate server[%d] failed and got: '%w'", index, err))
		}
	}
	return errors.NewAggregate(errs)
}

// SetDefaults set default values for frp client options.
func (o *Options) SetDefaults() {
	if o.Workspace == "" {
		o.Workspace = defaultWorkspace
	}
	if o.Servers == nil {
		o.Servers = make([]ServerConfig, 0)
	}
}

// AddFlags add related command line parameters
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Workspace, "frpc.workspace", o.Workspace, "The workspace for "+
		"frpc-service-provider. config file will be generated in this directory.")
}
