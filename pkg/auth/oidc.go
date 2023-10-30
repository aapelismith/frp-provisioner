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
	"context"
	"fmt"
	"github.com/fatedier/frp/pkg/msg"

	"github.com/samber/lo"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/frp-sigs/frp-provisioner/pkg/config"
)

type OidcAuthProvider struct {
	additionalAuthScopes []config.AuthScope

	tokenGenerator *clientcredentials.Config
}

func NewOidcAuthSetter(additionalAuthScopes []config.AuthScope, cfg config.AuthOIDCClientConfig) *OidcAuthProvider {
	eps := make(map[string][]string)
	for k, v := range cfg.AdditionalEndpointParams {
		eps[k] = []string{v}
	}

	if cfg.Audience != "" {
		eps["audience"] = []string{cfg.Audience}
	}

	tokenGenerator := &clientcredentials.Config{
		ClientID:       cfg.ClientID,
		ClientSecret:   cfg.ClientSecret,
		Scopes:         []string{cfg.Scope},
		TokenURL:       cfg.TokenEndpointURL,
		EndpointParams: eps,
	}

	return &OidcAuthProvider{
		additionalAuthScopes: additionalAuthScopes,
		tokenGenerator:       tokenGenerator,
	}
}

func (auth *OidcAuthProvider) generateAccessToken() (accessToken string, err error) {
	tokenObj, err := auth.tokenGenerator.Token(context.Background())
	if err != nil {
		return "", fmt.Errorf("couldn't generate OIDC token for login: %v", err)
	}
	return tokenObj.AccessToken, nil
}

func (auth *OidcAuthProvider) SetLogin(loginMsg *msg.Login) (err error) {
	loginMsg.PrivilegeKey, err = auth.generateAccessToken()
	return err
}

func (auth *OidcAuthProvider) SetPing(pingMsg *msg.Ping) (err error) {
	if !lo.Contains(auth.additionalAuthScopes, config.AuthScopeHeartBeats) {
		return nil
	}

	pingMsg.PrivilegeKey, err = auth.generateAccessToken()
	return err
}

func (auth *OidcAuthProvider) SetNewWorkConn(newWorkConnMsg *msg.NewWorkConn) (err error) {
	if !lo.Contains(auth.additionalAuthScopes, config.AuthScopeNewWorkConns) {
		return nil
	}

	newWorkConnMsg.PrivilegeKey, err = auth.generateAccessToken()
	return err
}
