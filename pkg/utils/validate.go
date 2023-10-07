package utils

import "fmt"

// ValidatePort checks that the network port is in range
func ValidatePort(port int, fieldPath string) error {
	if 0 <= port && port <= 65535 {
		return nil
	}
	return fmt.Errorf("%s: port number %d must be in the range 0..65535", fieldPath, port)
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
