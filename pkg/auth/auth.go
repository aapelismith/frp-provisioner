// Copyright 2020 guylewin, guy@lewin.co.il
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

package auth

import (
	"fmt"
	"github.com/fatedier/frp/pkg/msg"

	"github.com/aapelismith/frp-provisioner/pkg/config"
)

type Setter interface {
	SetLogin(*msg.Login) error
	SetPing(*msg.Ping) error
	SetNewWorkConn(*msg.NewWorkConn) error
}

func NewAuthSetter(cfg config.AuthClientConfig) (authProvider Setter) {
	switch cfg.Method {
	case config.AuthMethodToken:
		authProvider = NewTokenAuth(cfg.AdditionalScopes, cfg.Token)
	case config.AuthMethodOIDC:
		authProvider = NewOidcAuthSetter(cfg.AdditionalScopes, cfg.OIDC)
	default:
		panic(fmt.Sprintf("wrong method: '%s'", cfg.Method))
	}
	return authProvider
}
