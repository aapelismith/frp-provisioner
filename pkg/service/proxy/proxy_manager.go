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

package proxy

import (
	"context"
	"fmt"
	"github.com/aapelismith/frp-provisioner/pkg/service/event"
	"github.com/fatedier/frp/pkg/msg"
	"net"
	"reflect"
	"sync"

	"github.com/samber/lo"

	"github.com/aapelismith/frp-provisioner/pkg/config"
	"github.com/fatedier/frp/pkg/transport"
	"github.com/fatedier/frp/pkg/util/xlog"
)

type Manager struct {
	proxies        map[string]*Wrapper
	msgTransporter transport.MessageTransporter

	closed bool
	mu     sync.RWMutex

	clientCfg *config.ClientCommonConfig

	ctx context.Context
}

func NewManager(
	ctx context.Context,
	clientCfg *config.ClientCommonConfig,
	msgTransporter transport.MessageTransporter,
) *Manager {
	return &Manager{
		proxies:        make(map[string]*Wrapper),
		msgTransporter: msgTransporter,
		closed:         false,
		clientCfg:      clientCfg,
		ctx:            ctx,
	}
}

func (pm *Manager) StartProxy(name string, remoteAddr string, serverRespErr string) error {
	pm.mu.RLock()
	pxy, ok := pm.proxies[name]
	pm.mu.RUnlock()
	if !ok {
		return fmt.Errorf("proxy [%s] not found", name)
	}

	err := pxy.SetRunningStatus(remoteAddr, serverRespErr)
	if err != nil {
		return err
	}
	return nil
}

func (pm *Manager) Close() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	for _, pxy := range pm.proxies {
		pxy.Stop()
	}
	pm.proxies = make(map[string]*Wrapper)
}

func (pm *Manager) HandleWorkConn(name string, workConn net.Conn, m *msg.StartWorkConn) {
	pm.mu.RLock()
	pw, ok := pm.proxies[name]
	pm.mu.RUnlock()
	if ok {
		pw.InWorkConn(workConn, m)
	} else {
		_ = workConn.Close()
	}
}

func (pm *Manager) HandleEvent(payload interface{}) error {
	var m msg.Message
	switch e := payload.(type) {
	case *event.StartProxyPayload:
		m = e.NewProxyMsg
	case *event.CloseProxyPayload:
		m = e.CloseProxyMsg
	default:
		return event.ErrPayloadType
	}

	return pm.msgTransporter.Send(m)
}

func (pm *Manager) GetAllProxyStatus() []*WorkingStatus {
	ps := make([]*WorkingStatus, 0)
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	for _, pxy := range pm.proxies {
		ps = append(ps, pxy.GetStatus())
	}
	return ps
}

func (pm *Manager) Reload(proxyConfigures []config.ProxyConfigurer) {
	xl := xlog.FromContextSafe(pm.ctx)
	proxyConfigMap := lo.KeyBy(proxyConfigures, func(c config.ProxyConfigurer) string {
		return c.GetBaseConfig().Name
	})
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delPxyNames := make([]string, 0)
	for name, pxy := range pm.proxies {
		del := false
		cfg, ok := proxyConfigMap[name]
		if !ok || !reflect.DeepEqual(pxy.Cfg, cfg) {
			del = true
		}

		if del {
			delPxyNames = append(delPxyNames, name)
			delete(pm.proxies, name)
			pxy.Stop()
		}
	}
	if len(delPxyNames) > 0 {
		xl.Info("proxy removed: %s", delPxyNames)
	}

	addPxyNames := make([]string, 0)
	for _, cfg := range proxyConfigures {
		name := cfg.GetBaseConfig().Name
		if _, ok := pm.proxies[name]; !ok {
			pxy := NewWrapper(pm.ctx, cfg, pm.clientCfg, pm.HandleEvent, pm.msgTransporter)
			pm.proxies[name] = pxy
			addPxyNames = append(addPxyNames, name)

			pxy.Start()
		}
	}
	if len(addPxyNames) > 0 {
		xl.Info("proxy added: %s", addPxyNames)
	}
}
