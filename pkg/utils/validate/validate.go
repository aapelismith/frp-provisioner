package validate

import (
	"context"
	"fmt"
	frpclient "github.com/fatedier/frp/client"
	"github.com/fatedier/frp/pkg/auth"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/util/version"
	"os"
	runtime2 "runtime"
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

// ValidateClientCommonConfig validate and check common config
func ValidateClientCommonConfig(ctx context.Context, cfg *v1.ClientCommonConfig) error {
	_, err := validation.ValidateClientCommonConfig(cfg)
	if err != nil {
		return err
	}
	var (
		loginRespMsg msg.LoginResp
		logger       = log.FromContext(ctx)
		authSetter   = auth.NewAuthSetter(cfg.Auth)
	)
	connMgr := frpclient.NewConnectionManager(ctx, cfg)
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
		Os:        runtime2.GOOS,
		Arch:      runtime2.GOARCH,
		User:      cfg.User,
		Timestamp: time.Now().Unix(),
		PoolCount: cfg.Transport.PoolCount,
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

/*func main() {*/
//ctx, cancel := context.WithCancel(context.Background())
//defer cancel()
//
//cfg := v1.ClientCommonConfig{
//	Auth: v1.AuthClientConfig{
//		Method: "token",
//		Token:  "C5emntm*SSrd%35I%R4Z5lQ6I4Y4T$4H0h%eCtLF",
//	},
//	ServerAddr: "tunnel.kunstack.com",
//	ServerPort: 7000,
//}
//
//cfg.SetDefaults()
//
//warn, err := validation.ValidateClientCommonConfig(&cfg)
//if err != nil {
//	panic(err)
//}
//if warn != nil {
//	fmt.Printf("WARN %s", warn)
//}
//
//authSetter := auth.NewAuthSetter(cfg.Auth)
//
//cm := client.NewConnectionManager(ctx, &cfg)
//
//if err = cm.OpenConnection(); err != nil {
//	panic(err)
//}
//
//defer func() {
//	if err != nil {
//		cm.Close()
//	}
//}()
//
//conn, err := cm.Connect()
//if err != nil {
//	return
//}
//defer func() {
//	_ = conn.Close()
//}()
//
//loginMsg := &api.Login{
//	Arch:      runtime.GOARCH,
//	Os:        runtime.GOOS,
//	PoolCount: cfg.Transport.PoolCount,
//	User:      cfg.User,
//	Version:   version.Full(),
//	Timestamp: time.Now().Unix(),
//}
//
//if err := authSetter.SetLogin(loginMsg); err != nil {
//	panic(err)
//}
//
//if err = api.WriteMsg(conn, loginMsg); err != nil {
//	return
//}
//
//var loginRespMsg api.LoginResp
//_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
//if err = api.ReadMsgInto(conn, &loginRespMsg); err != nil {
//	return
//}
//_ = conn.SetReadDeadline(time.Time{})
//
//fmt.Printf("loginRespMsg=%+v", loginRespMsg)
//
//if loginRespMsg.Error != "" {
//	err = fmt.Errorf("%s", loginRespMsg.Error)
//	fmt.Printf("ERR %s", loginRespMsg.Error)
//	return
//}
//
//go func() {
//	encReader := crypto.NewReader(conn, []byte(cfg.Auth.Token))
//
//	for {
//		m, err := api.ReadMsg(encReader)
//		if err != nil {
//			if err == io.EOF {
//				fmt.Printf("read from control connection EOF")
//				return
//			}
//			fmt.Printf("read error: %v\n", err)
//			conn.Close()
//			return
//		}
//		fmt.Printf("recived %+v, type=%s\n", m, reflect.TypeOf(m))
//	}
//}()
//
//encWriter, err := crypto.NewWriter(conn, []byte(cfg.Auth.Token))
//if err != nil {
//	panic(err)
//}
//
//c := v1.UDPProxyConfig{
//	ProxyBaseConfig: v1.ProxyBaseConfig{
//		Name: "hello",
//		Type: "tcp",
//		ProxyBackend: v1.ProxyBackend{
//			LocalIP:   "localhost",
//			LocalPort: 8088,
//		},
//	},
//	RemotePort: 8088,
//}
//
//var newProxyMsg api.NewProxy
//c.MarshalToMsg(&newProxyMsg)
//
//if err = api.WriteMsg(encWriter, &newProxyMsg); err != nil {
//	panic(err)
//	return
//}
//
//time.Sleep(time.Second * 5)
//
//closeMsg := api.CloseProxy{
//	ProxyName: "hello",
//}
//if err = api.WriteMsg(encWriter, &closeMsg); err != nil {
//	panic(err)
//	return
//}
//select {}
//
//}
