// Copyright 2017 fatedier, fatedier@gmail.com
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

package service

import (
	"context"
	"github.com/aapelismith/frp-provisioner/pkg/auth"
	"github.com/aapelismith/frp-provisioner/pkg/log"
	"github.com/aapelismith/frp-provisioner/pkg/service/proxy"
	"github.com/fatedier/frp/pkg/msg"
	"go.uber.org/zap"
	"io"
	"net"
	"runtime/debug"
	"time"

	"github.com/fatedier/golib/control/shutdown"
	"github.com/fatedier/golib/crypto"
	"github.com/samber/lo"

	"github.com/aapelismith/frp-provisioner/pkg/config"
	"github.com/fatedier/frp/pkg/transport"
)

type Control struct {
	// service context
	ctx    context.Context
	logger *zap.SugaredLogger

	// Unique ID obtained from frps.
	// It should be attached to the login message when reconnecting.
	runID string

	// manage all proxies
	proxyConfigures []config.ProxyConfigurer
	pm              *proxy.Manager

	// control connection
	conn net.Conn

	cm *ConnectionManager

	// put a message in this channel to send it over control connection to server
	sendCh chan msg.Message

	// read from this channel to get the next message sent by server
	readCh chan msg.Message

	// goroutines can block by reading from this channel, it will be closed only in reader() when control connection is closed
	closedCh chan struct{}

	closedDoneCh chan struct{}

	// last time got the Pong message
	lastPong time.Time

	// The client configuration
	clientCfg *config.ClientCommonConfig

	readerShutdown     *shutdown.Shutdown
	writerShutdown     *shutdown.Shutdown
	msgHandlerShutdown *shutdown.Shutdown

	// sets authentication based on selected method
	authSetter auth.Setter

	msgTransporter transport.MessageTransporter
}

func NewControl(
	ctx context.Context, runID string,
	conn net.Conn, cm *ConnectionManager,
	clientCfg *config.ClientCommonConfig,
	proxyConfigures []config.ProxyConfigurer,
	authSetter auth.Setter,
) *Control {
	ctl := &Control{
		ctx:                ctx,
		logger:             log.FromContext(ctx).Sugar(),
		runID:              runID,
		conn:               conn,
		cm:                 cm,
		proxyConfigures:    proxyConfigures,
		sendCh:             make(chan msg.Message, 100),
		readCh:             make(chan msg.Message, 100),
		closedCh:           make(chan struct{}),
		closedDoneCh:       make(chan struct{}),
		clientCfg:          clientCfg,
		readerShutdown:     shutdown.New(),
		writerShutdown:     shutdown.New(),
		msgHandlerShutdown: shutdown.New(),
		authSetter:         authSetter,
	}
	ctl.msgTransporter = transport.NewMessageTransporter(ctl.sendCh)
	ctl.pm = proxy.NewManager(ctl.ctx, clientCfg, ctl.msgTransporter)
	return ctl
}

func (ctl *Control) Run() {
	go ctl.worker()

	// start all proxies
	ctl.pm.Reload(ctl.proxyConfigures)
}

func (ctl *Control) HandleReqWorkConn(_ *msg.ReqWorkConn) {
	workConn, err := ctl.connectServer()
	if err != nil {
		ctl.logger.Warn("start new connection to server error: %v", err)
		return
	}

	m := &msg.NewWorkConn{
		RunID: ctl.runID,
	}
	if err = ctl.authSetter.SetNewWorkConn(m); err != nil {
		ctl.logger.Warn("error during NewWorkConn authentication: %v", err)
		return
	}
	if err = msg.WriteMsg(workConn, m); err != nil {
		ctl.logger.Warn("work connection write to server error: %v", err)
		_ = workConn.Close()
		return
	}

	var startMsg msg.StartWorkConn
	if err = msg.ReadMsgInto(workConn, &startMsg); err != nil {
		ctl.logger.Errorf("work connection closed before response StartWorkConn message: %v", err)
		_ = workConn.Close()
		return
	}
	if startMsg.Error != "" {
		ctl.logger.Error("StartWorkConn contains error: %s", startMsg.Error)
		_ = workConn.Close()
		return
	}

	// dispatch this work connection to related proxy
	ctl.pm.HandleWorkConn(startMsg.ProxyName, workConn, &startMsg)
}

func (ctl *Control) HandleNewProxyResp(inMsg *msg.NewProxyResp) {
	// Server will return NewProxyResp message to each NewProxy message.
	// Start a new proxy handler if no error got
	err := ctl.pm.StartProxy(inMsg.ProxyName, inMsg.RemoteAddr, inMsg.Error)
	if err != nil {
		ctl.logger.Warn("[%s] start error: %v", inMsg.ProxyName, err)
	} else {
		ctl.logger.Info("[%s] start proxy success", inMsg.ProxyName)
	}
}

func (ctl *Control) HandleNatHoleResp(inMsg *msg.NatHoleResp) {
	// Dispatch the NatHoleResp message to the related proxy.
	ok := ctl.msgTransporter.DispatchWithType(inMsg, msg.TypeNameNatHoleResp, inMsg.TransactionID)
	if !ok {
		ctl.logger.Error("dispatch NatHoleResp message to related proxy error")
	}
}

func (ctl *Control) Close() error {
	return ctl.GracefulClose(0)
}

func (ctl *Control) GracefulClose(d time.Duration) error {
	ctl.pm.Close()

	time.Sleep(d)

	_ = ctl.conn.Close()
	_ = ctl.cm.Close()
	return nil
}

// ClosedDoneCh returns a channel that will be closed after all resources are released
func (ctl *Control) ClosedDoneCh() <-chan struct{} {
	return ctl.closedDoneCh
}

// connectServer return a new connection to frps
func (ctl *Control) connectServer() (conn net.Conn, err error) {
	return ctl.cm.Connect()
}

// reader read all messages from frps and send to readCh
func (ctl *Control) reader() {
	defer func() {
		if err := recover(); err != nil {
			ctl.logger.Error("panic error: %v", err)
			ctl.logger.Error(string(debug.Stack()))
		}
	}()

	defer ctl.readerShutdown.Done()
	defer close(ctl.closedCh)

	encReader := crypto.NewReader(ctl.conn, []byte(ctl.clientCfg.Auth.Token))
	for {
		m, err := msg.ReadMsg(encReader)
		if err != nil {
			if err == io.EOF {
				ctl.logger.Debug("read from control connection EOF")
				return
			}
			ctl.logger.Warn("read error: %v", err)
			_ = ctl.conn.Close()
			return
		}
		ctl.readCh <- m
	}
}

// writer writes messages got from sendCh to frps
func (ctl *Control) writer() {
	xl := ctl.logger
	defer ctl.writerShutdown.Done()
	encWriter, err := crypto.NewWriter(ctl.conn, []byte(ctl.clientCfg.Auth.Token))
	if err != nil {
		xl.Error("crypto new writer error: %v", err)
		_ = ctl.conn.Close()
		return
	}
	for {
		m, ok := <-ctl.sendCh
		if !ok {
			xl.Info("control writer is closing")
			return
		}

		if err := msg.WriteMsg(encWriter, m); err != nil {
			xl.Warn("write message to control connection error: %v", err)
			return
		}
	}
}

// msgHandler handles all channel events and performs corresponding operations.
func (ctl *Control) msgHandler() {
	xl := ctl.logger
	defer func() {
		if err := recover(); err != nil {
			xl.Error("panic error: %v", err)
			xl.Error(string(debug.Stack()))
		}
	}()
	defer ctl.msgHandlerShutdown.Done()

	var hbSendCh <-chan time.Time
	// TODO(fatedier): disable heartbeat if TCPMux is enabled.
	// Just keep it here to keep compatible with old version frps.
	if ctl.clientCfg.Transport.HeartbeatInterval > 0 {
		hbSend := time.NewTicker(time.Duration(ctl.clientCfg.Transport.HeartbeatInterval) * time.Second)
		defer hbSend.Stop()
		hbSendCh = hbSend.C
	}

	var hbCheckCh <-chan time.Time
	// Check heartbeat timeout only if TCPMux is not enabled and users don't disable heartbeat feature.
	if ctl.clientCfg.Transport.HeartbeatInterval > 0 && ctl.clientCfg.Transport.HeartbeatTimeout > 0 &&
		!lo.FromPtr(ctl.clientCfg.Transport.TCPMux) {
		hbCheck := time.NewTicker(time.Second)
		defer hbCheck.Stop()
		hbCheckCh = hbCheck.C
	}

	ctl.lastPong = time.Now()
	for {
		select {
		case <-hbSendCh:
			// send heartbeat to server
			xl.Debug("send heartbeat to server")
			pingMsg := &msg.Ping{}
			if err := ctl.authSetter.SetPing(pingMsg); err != nil {
				xl.Warn("error during ping authentication: %v", err)
				return
			}
			ctl.sendCh <- pingMsg
		case <-hbCheckCh:
			if time.Since(ctl.lastPong) > time.Duration(ctl.clientCfg.Transport.HeartbeatTimeout)*time.Second {
				xl.Warn("heartbeat timeout")
				// let reader() stop
				_ = ctl.conn.Close()
				return
			}
		case rawMsg, ok := <-ctl.readCh:
			if !ok {
				return
			}

			switch m := rawMsg.(type) {
			case *msg.ReqWorkConn:
				go ctl.HandleReqWorkConn(m)
			case *msg.NewProxyResp:
				ctl.HandleNewProxyResp(m)
			case *msg.NatHoleResp:
				ctl.HandleNatHoleResp(m)
			case *msg.Pong:
				if m.Error != "" {
					xl.Error("Pong contains error: %s", m.Error)
					_ = ctl.conn.Close()
					return
				}
				ctl.lastPong = time.Now()
				xl.Debug("receive heartbeat from server")
			}
		}
	}
}

// If controller is notified by closedCh, reader and writer and handler will exit
func (ctl *Control) worker() {
	go ctl.msgHandler()
	go ctl.reader()
	go ctl.writer()

	<-ctl.closedCh
	// close related channels and wait until other goroutines done
	close(ctl.readCh)
	ctl.readerShutdown.WaitDone()
	ctl.msgHandlerShutdown.WaitDone()

	close(ctl.sendCh)
	ctl.writerShutdown.WaitDone()

	ctl.pm.Close()

	close(ctl.closedDoneCh)
	_ = ctl.cm.Close()
}

func (ctl *Control) ReloadConf(poxyConfigs []config.ProxyConfigurer) error {
	ctl.pm.Reload(poxyConfigs)
	return nil
}
